// Copyright (c) 2021 PaddlePaddle Authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mpc

import (
	"errors"
	"io/ioutil"
	"strconv"
	"testing"
	"time"

	vl_common "github.com/PaddlePaddle/PaddleDTX/dai/crypto/vl/common"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/predictor"
	"github.com/PaddlePaddle/PaddleDTX/dai/mpc/trainer"
	"github.com/PaddlePaddle/PaddleDTX/dai/p2p"
	pbCom "github.com/PaddlePaddle/PaddleDTX/dai/protos/common"
	pb "github.com/PaddlePaddle/PaddleDTX/dai/protos/mpc"
)

var (
	testP2P = p2p.NewP2P()
	mpc1    *mpc
	mpc2    *mpc
)

type testModelHolder struct {
}

func (tmh *testModelHolder) SaveModel(result *pbCom.TrainTaskResult) error {
	return nil
}

func (tmh *testModelHolder) SavePredictOut(result *pbCom.PredictTaskResult) error {
	return nil
}

func TestMpc(t *testing.T) {
	mh := &testModelHolder{}

	config := Config{
		Address:          "127.0.0.1:8080",
		TrainTaskLimit:   5,
		PredictTaskLimit: 10,
		RpcTimeout:       3,
	}
	testMpc := StartMpc(mh, testP2P, config)

	// test StartTask and StopTask
	numStartT := 6
	var startTaskReqs []*pbCom.StartTaskRequest
	for i := 0; i < numStartT; i++ {
		startTaskReqs = append(startTaskReqs, &pbCom.StartTaskRequest{
			TaskID: "Test-Mpc" + strconv.Itoa(i),
			File:   []byte("Hello,world"),
			Hosts:  []string{"127.0.0.1:8081"},
			Params: &pbCom.TaskParams{
				Algo:        pbCom.Algorithm_LINEAR_REGRESSION_VL,
				TaskType:    pbCom.TaskType_LEARN,
				TrainParams: &pbCom.TrainParams{},
				ModelParams: &pbCom.TrainModels{},
			},
		})
	}

	for i := 0; i < numStartT*2; i++ {
		startTaskReqs = append(startTaskReqs, &pbCom.StartTaskRequest{
			TaskID: "Test-Mpc" + strconv.Itoa(i),
			File:   []byte("Hello,world"),
			Hosts:  []string{"127.0.0.1:8081"},
			Params: &pbCom.TaskParams{
				Algo:        pbCom.Algorithm_LOGIC_REGRESSION_VL,
				TaskType:    pbCom.TaskType_PREDICT,
				TrainParams: &pbCom.TrainParams{},
				ModelParams: &pbCom.TrainModels{},
			},
		})
	}

	for i := 0; i < numStartT*3; i++ {
		go func(req *pbCom.StartTaskRequest) {
			err := testMpc.StartTask(req)
			if err != nil {
				t.Logf("failed StartTask[%v], error: %s", req, err.Error())
			} else {
				t.Logf("StartTask[%v]", req)
			}
		}(startTaskReqs[i])
	}

	time.Sleep(3 * time.Second)

	// test StopTask
	stopTaskReq := &pbCom.StopTaskRequest{
		TaskID: "Test-Mpc-StopTask",
	}
	err := testMpc.StopTask(stopTaskReq)
	if err != nil {
		t.Logf("failed StopTask[%v], error: %s", stopTaskReq, err.Error())
	} else {
		t.Logf("StopTask[%v]", stopTaskReq)
	}

	// test Train
	trainReq := &pb.TrainRequest{
		TaskID:  "TrainRequest-test",
		Algo:    pbCom.Algorithm_LINEAR_REGRESSION_VL,
		Payload: []byte("Hello! World!"),
	}
	respT, err := testMpc.Train(trainReq)
	if err != nil {
		t.Logf("Train request[%v], response[%v], error: %v", trainReq, respT, err.Error())
	} else {
		t.Logf("Train request[%v], response[%v]", trainReq, respT)
	}

	// test Predict
	predictReq := &pb.PredictRequest{
		TaskID:  "PredictRequest-test",
		Algo:    pbCom.Algorithm_LINEAR_REGRESSION_VL,
		Payload: []byte("Hello! World!"),
	}
	respP, err := testMpc.Predict(predictReq)
	if err != nil {
		t.Logf("Predict request[%v], response[%v], error: %v", predictReq, respP, err.Error())
	} else {
		t.Logf("Predict request[%v], response[%v]", predictReq, respP)
	}

	// test Stop
	testMpc.Stop()
}

func TestValidate(t *testing.T) {
	mh := &testModelHolder{}

	config := Config{
		Address:          "127.0.0.1:8080",
		TrainTaskLimit:   5,
		PredictTaskLimit: 10,
		RpcTimeout:       3,
	}
	testMpc := StartMpc(mh, testP2P, config)

	// test Validate
	predRes := &pbCom.PredictTaskResult{
		TaskID:   "uuid_0_predict_Eva",
		Success:  true,
		Outcomes: []byte("PredictionOut"),
	}
	validateReq := &pb.ValidateRequest{
		TaskID:        "uuid",
		From:          pb.Evaluator_NORMAL,
		FoldIdx:       0,
		PredictResult: predRes,
	}
	err := testMpc.Validate(validateReq)
	if err != nil {
		t.Logf("Validate request[%v], error: %v", validateReq, err.Error())
	} else {
		t.Logf("Predict request[%v], no error", validateReq)
	}

	predRes = &pbCom.PredictTaskResult{
		TaskID:   "uuid_0_predict_LEv",
		Success:  true,
		Outcomes: []byte("PredictionOut"),
	}

	validateReq = &pb.ValidateRequest{
		TaskID:        "uuid",
		From:          pb.Evaluator_LIVE,
		FoldIdx:       0,
		PredictResult: predRes,
	}
	err = testMpc.Validate(validateReq)
	if err != nil {
		t.Logf("Validate request[%v], error: %v", validateReq, err.Error())
	} else {
		t.Logf("Predict request[%v], no error", validateReq)
	}

	testMpc.Stop()
}

type rpc struct {
	reqTrainC  chan *pb.TrainRequest
	respTrainC chan *pb.TrainResponse

	reqPredC  chan *pb.PredictRequest
	respPredC chan *pb.PredictResponse

	t *testing.T
}

func (r *rpc) StepTrainWithRetry(req *pb.TrainRequest, peerName string, times int, inteSec int64) (*pb.TrainResponse, error) {
	//r.t.Logf("Step training message[%v]", req)
	r.reqTrainC <- req
	resp := <-r.respTrainC
	if resp != nil {
		return resp, nil
	}

	return nil, errors.New("test response error")
}

func (r *rpc) StepTrain(req *pb.TrainRequest, peerName string) (*pb.TrainResponse, error) {
	//r.t.Logf("Step training message[%v]", req)
	r.reqTrainC <- req
	resp := <-r.respTrainC
	if resp != nil {
		return resp, nil
	}

	return nil, errors.New("test response error")
}

func (r *rpc) StepPredictWithRetry(req *pb.PredictRequest, peerName string, times int, inteSec int64) (*pb.PredictResponse, error) {
	r.t.Logf("Step prediction message[%v]", req)
	r.reqPredC <- req
	resp := <-r.respPredC
	if resp != nil {
		return resp, nil
	}
	return nil, errors.New("test response error")
}

func (r *rpc) StepPredict(req *pb.PredictRequest, peerName string) (*pb.PredictResponse, error) {
	r.t.Logf("Step prediction message[%v]", req)
	r.reqPredC <- req
	resp := <-r.respPredC
	if resp != nil {
		return resp, nil
	}
	return nil, errors.New("test response error")
}

type modelHolder struct {
	t              *testing.T
	trainResultC   chan *pbCom.TrainTaskResult
	predictResultC chan *pbCom.PredictTaskResult
}

func (mh *modelHolder) SaveModel(result *pbCom.TrainTaskResult) error {
	if result.Success {
		mh.t.Logf("Get model and save")
		mh.trainResultC <- result
	} else {
		mh.t.Logf("training failed, and reason is %s.", result.ErrMsg)
		mh.trainResultC <- nil
	}
	return nil
}

func (mh *modelHolder) SavePredictOut(result *pbCom.PredictTaskResult) error {
	if result.Success {
		mh.t.Logf("Get prediction outcome[%v] and save", result)
		mh.predictResultC <- result
	} else {
		mh.t.Logf("Get prediction outcome failed, and reason is %s.", result.ErrMsg)
		mh.predictResultC <- nil
	}

	return nil
}

func TestEvaluRegressionRandomSplit(t *testing.T) {
	//initiate mpc instance for party1
	var reqTC1 = make(chan *pb.TrainRequest)
	var respTC1 = make(chan *pb.TrainResponse)
	var reqPC1 = make(chan *pb.PredictRequest)
	var respPC1 = make(chan *pb.PredictResponse)
	var trainResultC1 = make(chan *pbCom.TrainTaskResult)
	var predictResultC1 = make(chan *pbCom.PredictTaskResult)

	rpc1 := &rpc{
		reqTrainC:  reqTC1,
		respTrainC: respTC1,

		reqPredC:  reqPC1,
		respPredC: respPC1,

		t: t,
	}

	mh1 := &modelHolder{
		trainResultC:   trainResultC1,
		predictResultC: predictResultC1,
		t:              t,
	}

	mpc1 = &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}

	conf1 := Config{
		Address:          ":8080",
		TrainTaskLimit:   1000,
		PredictTaskLimit: 2000,
		RpcTimeout:       time.Duration(3),
	}

	trainCallback1 := TrainCallBack{ModelHolder: mh1, Mpc: mpc1}
	mpc1.trainer = trainer.NewTrainer(conf1.Address, rpc1, &trainCallback1, conf1.TrainTaskLimit)

	predictCallBack1 := PredictCallBack{ModelHolder: mh1, Mpc: mpc1}
	mpc1.predictor = predictor.NewPredictor(conf1.Address, rpc1, &predictCallBack1, conf1.PredictTaskLimit)

	//initiate mpc instance for party2
	var reqTC2 = make(chan *pb.TrainRequest)
	var respTC2 = make(chan *pb.TrainResponse)
	var reqPC2 = make(chan *pb.PredictRequest)
	var respPC2 = make(chan *pb.PredictResponse)
	var trainResultC2 = make(chan *pbCom.TrainTaskResult)
	var predictResultC2 = make(chan *pbCom.PredictTaskResult)

	rpc2 := &rpc{
		reqTrainC:  reqTC2,
		respTrainC: respTC2,

		reqPredC:  reqPC2,
		respPredC: respPC2,

		t: t,
	}

	mh2 := &modelHolder{
		trainResultC:   trainResultC2,
		predictResultC: predictResultC2,
		t:              t,
	}

	mpc2 = &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}

	conf2 := Config{
		Address:          ":8081",
		TrainTaskLimit:   1000,
		PredictTaskLimit: 2000,
		RpcTimeout:       time.Duration(3),
	}

	trainCallback2 := TrainCallBack{ModelHolder: mh2, Mpc: mpc2}
	mpc2.trainer = trainer.NewTrainer(conf2.Address, rpc2, &trainCallback2, conf2.TrainTaskLimit)

	predictCallBack2 := PredictCallBack{ModelHolder: mh2, Mpc: mpc2}
	mpc2.predictor = predictor.NewPredictor(conf2.Address, rpc2, &predictCallBack2, conf2.PredictTaskLimit)

	// prepare requests
	trainFile1 := "./testdata/vl/linear_boston_housing/dataA.csv"

	samplesFile1, err := ioutil.ReadFile(trainFile1)
	checkErr(err, t)
	req1 := &pbCom.StartTaskRequest{
		TaskID: "TestEvaRegMpc",
		File:   samplesFile1,
		Hosts:  []string{":8081"},
		Params: &pbCom.TaskParams{
			Algo:     pbCom.Algorithm_LINEAR_REGRESSION_VL,
			TaskType: pbCom.TaskType_LEARN,
			TrainParams: &pbCom.TrainParams{
				Label:     "MEDV",
				RegMode:   0,
				RegParam:  0.1,
				Alpha:     0.1,
				Amplitude: 0.0001,
				Accuracy:  10,
				IsTagPart: false,
				IdName:    "id",
				BatchSize: 4,
			},
			EvalParams: &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule_ErRandomSplit,
				RandomSplit: &pbCom.RandomSplit{
					PercentLO: 10,
				},
			},
		},
	}

	trainFile2 := "./testdata/vl/linear_boston_housing/dataB.csv"
	samplesFile2, err := ioutil.ReadFile(trainFile2)
	checkErr(err, t)
	req2 := &pbCom.StartTaskRequest{
		TaskID: "TestEvaRegMpc",
		File:   samplesFile2,
		Hosts:  []string{":8080"},
		Params: &pbCom.TaskParams{
			Algo:     pbCom.Algorithm_LINEAR_REGRESSION_VL,
			TaskType: pbCom.TaskType_LEARN,
			TrainParams: &pbCom.TrainParams{
				Label:     "MEDV",
				RegMode:   0,
				RegParam:  0.1,
				Alpha:     0.1,
				Amplitude: 0.0001,
				Accuracy:  10,
				IsTagPart: true,
				IdName:    "id",
				BatchSize: 4,
			},
			EvalParams: &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule_ErRandomSplit,
				RandomSplit: &pbCom.RandomSplit{
					PercentLO: 10,
				},
			},
		},
	}

	// start test
	go mpc1.run()
	go mpc2.run()
	t.Log("Run Mpcs and wait a moment")
	time.Sleep(1 * time.Second)

	go func() {
		t.Log("Mpc1.StartTask")
		err = mpc1.StartTask(req1)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log("Mpc1.StartTask successfully")
	}()

	go func() {
		t.Log("Mpc2.StartTask")
		err = mpc2.StartTask(req2)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log("Mpc2.StartTask successfully")
	}()

	time.Sleep(2 * time.Second)

	// train and evaluate
	var done = make(chan int)
	var stop = make(chan int, 1)
	var stopped1 bool
	var stopped2 bool

	isDone := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	for {
		select {
		case reqTRecv2 := <-reqTC1:
			respTSend2, err := mpc2.Train(reqTRecv2)
			if err != nil {
				t.Logf("Mpc2.Train err: %s", err.Error())
			}
			respTC1 <- respTSend2
		case trainResult1 := <-trainResultC1:
			if trainResult1 == nil { //failed
				t.Logf("mpc1 train model failed")
			} else {
				model1, _ := vl_common.TrainModelsFromBytes(trainResult1.Model)
				t.Logf("mpc1 train and evaluate model successfully, and model is[%v], and evaluation result is[%v]", model1, trainResult1.EvalMetricScores)
			}
			stopped1 = true
			if stopped1 && stopped2 {
				stop <- 1
			}

		case reqTRecv1 := <-reqTC2:
			respTSend1, err := mpc1.Train(reqTRecv1)
			if err != nil {
				t.Logf("Mpc1.Train err: %s", err.Error())
			}
			respTC2 <- respTSend1
		case trainResult2 := <-trainResultC2:
			if trainResult2 == nil { //failed
				t.Logf("mpc2 train model failed")
			} else {
				model2, _ := vl_common.TrainModelsFromBytes(trainResult2.Model)
				t.Logf("mpc2 train and evaluate model successfully, and model is[%v], and evaluation result is[%v]", model2, trainResult2.EvalMetricScores)
			}
			stopped2 = true
			if stopped1 && stopped2 {
				stop <- 1
			}
		case reqPRecv2 := <-reqPC1:
			respPSend2, err := mpc2.Predict(reqPRecv2)
			if err != nil {
				t.Logf("Mpc2.Predict err: %s", err.Error())
			}
			respPC1 <- respPSend2
		case <-predictResultC1:
			t.Error("Something wrong happened, because mpc1 shouldn't return prediction result.")
			t.FailNow()

		case reqPRecv1 := <-reqPC2:
			respPSend1, err := mpc1.Predict(reqPRecv1)
			if err != nil {
				t.Logf("Mpc1.Predict err: %s", err.Error())
			}
			respPC2 <- respPSend1
		case <-predictResultC2:
			t.Error("Something wrong happened, because mpc2 shouldn't return prediction result.")
			t.FailNow()

		case <-stop:
			close(done)
		}

		if isDone() {
			break
		}
	}
}

func TestEvaluRegressionCV(t *testing.T) {
	//initiate mpc instance for party1
	var reqTC1 = make(chan *pb.TrainRequest)
	var respTC1 = make(chan *pb.TrainResponse)
	var reqPC1 = make(chan *pb.PredictRequest)
	var respPC1 = make(chan *pb.PredictResponse)
	var trainResultC1 = make(chan *pbCom.TrainTaskResult)
	var predictResultC1 = make(chan *pbCom.PredictTaskResult)

	rpc1 := &rpc{
		reqTrainC:  reqTC1,
		respTrainC: respTC1,

		reqPredC:  reqPC1,
		respPredC: respPC1,

		t: t,
	}

	mh1 := &modelHolder{
		trainResultC:   trainResultC1,
		predictResultC: predictResultC1,
		t:              t,
	}

	mpc1 = &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}

	conf1 := Config{
		Address:          ":8080",
		TrainTaskLimit:   1000,
		PredictTaskLimit: 2000,
		RpcTimeout:       time.Duration(3),
	}

	trainCallback1 := TrainCallBack{ModelHolder: mh1, Mpc: mpc1}
	mpc1.trainer = trainer.NewTrainer(conf1.Address, rpc1, &trainCallback1, conf1.TrainTaskLimit)

	predictCallBack1 := PredictCallBack{ModelHolder: mh1, Mpc: mpc1}
	mpc1.predictor = predictor.NewPredictor(conf1.Address, rpc1, &predictCallBack1, conf1.PredictTaskLimit)

	//initiate mpc instance for party2
	var reqTC2 = make(chan *pb.TrainRequest)
	var respTC2 = make(chan *pb.TrainResponse)
	var reqPC2 = make(chan *pb.PredictRequest)
	var respPC2 = make(chan *pb.PredictResponse)
	var trainResultC2 = make(chan *pbCom.TrainTaskResult)
	var predictResultC2 = make(chan *pbCom.PredictTaskResult)

	rpc2 := &rpc{
		reqTrainC:  reqTC2,
		respTrainC: respTC2,

		reqPredC:  reqPC2,
		respPredC: respPC2,

		t: t,
	}

	mh2 := &modelHolder{
		trainResultC:   trainResultC2,
		predictResultC: predictResultC2,
		t:              t,
	}

	mpc2 = &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}

	conf2 := Config{
		Address:          ":8081",
		TrainTaskLimit:   1000,
		PredictTaskLimit: 2000,
		RpcTimeout:       time.Duration(3),
	}

	trainCallback2 := TrainCallBack{ModelHolder: mh2, Mpc: mpc2}
	mpc2.trainer = trainer.NewTrainer(conf2.Address, rpc2, &trainCallback2, conf2.TrainTaskLimit)

	predictCallBack2 := PredictCallBack{ModelHolder: mh2, Mpc: mpc2}
	mpc2.predictor = predictor.NewPredictor(conf2.Address, rpc2, &predictCallBack2, conf2.PredictTaskLimit)

	// prepare requests
	trainFile1 := "./testdata/vl/linear_boston_housing/dataA.csv"

	samplesFile1, err := ioutil.ReadFile(trainFile1)
	checkErr(err, t)
	req1 := &pbCom.StartTaskRequest{
		TaskID: "TestEvaRegMpc",
		File:   samplesFile1,
		Hosts:  []string{":8081"},
		Params: &pbCom.TaskParams{
			Algo:     pbCom.Algorithm_LINEAR_REGRESSION_VL,
			TaskType: pbCom.TaskType_LEARN,
			TrainParams: &pbCom.TrainParams{
				Label:     "MEDV",
				RegMode:   0,
				RegParam:  0.1,
				Alpha:     0.1,
				Amplitude: 0.0001,
				Accuracy:  10,
				IsTagPart: false,
				IdName:    "id",
				BatchSize: 4,
			},
			EvalParams: &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule_ErCrossVal,
				Cv: &pbCom.CrossVal{
					Folds:   10,
					Shuffle: true,
				},
			},
		},
	}

	trainFile2 := "./testdata/vl/linear_boston_housing/dataB.csv"
	samplesFile2, err := ioutil.ReadFile(trainFile2)
	checkErr(err, t)
	req2 := &pbCom.StartTaskRequest{
		TaskID: "TestEvaRegMpc",
		File:   samplesFile2,
		Hosts:  []string{":8080"},
		Params: &pbCom.TaskParams{
			Algo:     pbCom.Algorithm_LINEAR_REGRESSION_VL,
			TaskType: pbCom.TaskType_LEARN,
			TrainParams: &pbCom.TrainParams{
				Label:     "MEDV",
				RegMode:   0,
				RegParam:  0.1,
				Alpha:     0.1,
				Amplitude: 0.0001,
				Accuracy:  10,
				IsTagPart: true,
				IdName:    "id",
				BatchSize: 4,
			},
			EvalParams: &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule_ErCrossVal,
				Cv: &pbCom.CrossVal{
					Folds:   10,
					Shuffle: true,
				},
			},
		},
	}

	// start test
	go mpc1.run()
	go mpc2.run()
	t.Log("Run Mpcs and wait a moment")
	time.Sleep(1 * time.Second)

	go func() {
		t.Log("Mpc1.StartTask")
		err = mpc1.StartTask(req1)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log("Mpc1.StartTask successfully")
	}()

	go func() {
		t.Log("Mpc2.StartTask")
		err = mpc2.StartTask(req2)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log("Mpc2.StartTask successfully")
	}()

	time.Sleep(2 * time.Second)

	// train and evaluate
	var done = make(chan int)
	var stop = make(chan int, 1)
	var stopped1 bool
	var stopped2 bool

	isDone := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	for {
		select {
		case reqTRecv2 := <-reqTC1:
			respTSend2, err := mpc2.Train(reqTRecv2)
			if err != nil {
				t.Logf("Mpc2.Train err: %s", err.Error())
			}
			respTC1 <- respTSend2
		case trainResult1 := <-trainResultC1:
			if trainResult1 == nil { //failed
				t.Logf("mpc1 train model failed")
			} else {
				model1, _ := vl_common.TrainModelsFromBytes(trainResult1.Model)
				t.Logf("mpc1 train and evaluate model successfully, and model is[%v], and evaluation result is[%v]", model1, trainResult1.EvalMetricScores)
			}
			stopped1 = true
			if stopped1 && stopped2 {
				stop <- 1
			}

		case reqTRecv1 := <-reqTC2:
			respTSend1, err := mpc1.Train(reqTRecv1)
			if err != nil {
				t.Logf("Mpc1.Train err: %s", err.Error())
			}
			respTC2 <- respTSend1
		case trainResult2 := <-trainResultC2:
			if trainResult2 == nil { //failed
				t.Logf("mpc2 train model failed")
			} else {
				model2, _ := vl_common.TrainModelsFromBytes(trainResult2.Model)
				t.Logf("mpc2 train and evaluate model successfully, and model is[%v], and evaluation result is[%v]", model2, trainResult2.EvalMetricScores)
			}
			stopped2 = true
			if stopped1 && stopped2 {
				stop <- 1
			}
		case reqPRecv2 := <-reqPC1:
			respPSend2, err := mpc2.Predict(reqPRecv2)
			if err != nil {
				t.Logf("Mpc2.Predict err: %s", err.Error())
			}
			respPC1 <- respPSend2
		case <-predictResultC1:
			t.Error("Something wrong happened, because mpc1 shouldn't return prediction result.")
			t.FailNow()

		case reqPRecv1 := <-reqPC2:
			respPSend1, err := mpc1.Predict(reqPRecv1)
			if err != nil {
				t.Logf("Mpc1.Predict err: %s", err.Error())
			}
			respPC2 <- respPSend1
		case <-predictResultC2:
			t.Error("Something wrong happened, because mpc2 shouldn't return prediction result.")
			t.FailNow()

		case <-stop:
			close(done)
		}

		if isDone() {
			break
		}
	}
}

/*
func TestEvaluRegressionLoo(t *testing.T) {
	//initiate mpc instance for party1
	var reqTC1 = make(chan *pb.TrainRequest)
	var respTC1 = make(chan *pb.TrainResponse)
	var reqPC1 = make(chan *pb.PredictRequest)
	var respPC1 = make(chan *pb.PredictResponse)
	var trainResultC1 = make(chan *pbCom.TrainTaskResult)
	var predictResultC1 = make(chan *pbCom.PredictTaskResult)

	rpc1 := &rpc{
		reqTrainC:  reqTC1,
		respTrainC: respTC1,

		reqPredC:  reqPC1,
		respPredC: respPC1,

		t: t,
	}

	mh1 := &modelHolder{
		trainResultC:   trainResultC1,
		predictResultC: predictResultC1,
		t:              t,
	}

	mpc1 = &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}

	conf1 := Config{
		Address:          ":8080",
		TrainTaskLimit:   1000,
		PredictTaskLimit: 2000,
		RpcTimeout:       time.Duration(3),
	}

	trainCallback1 := TrainCallBack{ModelHolder: mh1, Mpc: mpc1}
	mpc1.trainer = trainer.NewTrainer(conf1.Address, rpc1, &trainCallback1, conf1.TrainTaskLimit)

	predictCallBack1 := PredictCallBack{ModelHolder: mh1, Mpc: mpc1}
	mpc1.predictor = predictor.NewPredictor(conf1.Address, rpc1, &predictCallBack1, conf1.PredictTaskLimit)

	//initiate mpc instance for party2
	var reqTC2 = make(chan *pb.TrainRequest)
	var respTC2 = make(chan *pb.TrainResponse)
	var reqPC2 = make(chan *pb.PredictRequest)
	var respPC2 = make(chan *pb.PredictResponse)
	var trainResultC2 = make(chan *pbCom.TrainTaskResult)
	var predictResultC2 = make(chan *pbCom.PredictTaskResult)

	rpc2 := &rpc{
		reqTrainC:  reqTC2,
		respTrainC: respTC2,

		reqPredC:  reqPC2,
		respPredC: respPC2,

		t: t,
	}

	mh2 := &modelHolder{
		trainResultC:   trainResultC2,
		predictResultC: predictResultC2,
		t:              t,
	}

	mpc2 = &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}

	conf2 := Config{
		Address:          ":8081",
		TrainTaskLimit:   1000,
		PredictTaskLimit: 2000,
		RpcTimeout:       time.Duration(3),
	}

	trainCallback2 := TrainCallBack{ModelHolder: mh2, Mpc: mpc2}
	mpc2.trainer = trainer.NewTrainer(conf2.Address, rpc2, &trainCallback2, conf2.TrainTaskLimit)

	predictCallBack2 := PredictCallBack{ModelHolder: mh2, Mpc: mpc2}
	mpc2.predictor = predictor.NewPredictor(conf2.Address, rpc2, &predictCallBack2, conf2.PredictTaskLimit)

	// prepare requests
	trainFile1 := "./testdata/vl/linear_boston_housing/train_dataA.csv"

	samplesFile1, err := ioutil.ReadFile(trainFile1)
	checkErr(err, t)
	req1 := &pbCom.StartTaskRequest{
		TaskID: "TestEvaRegMpc",
		File:   samplesFile1,
		Hosts:  []string{":8081"},
		Params: &pbCom.TaskParams{
			Algo:     pbCom.Algorithm_LINEAR_REGRESSION_VL,
			TaskType: pbCom.TaskType_LEARN,
			TrainParams: &pbCom.TrainParams{
				Label:     "MEDV",
				RegMode:   0,
				RegParam:  0.1,
				Alpha:     0.1,
				Amplitude: 0.0001,
				Accuracy:  10,
				IsTagPart: false,
				IdName:    "id",
				BatchSize: 4,
			},
			EvalParams: &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule_ErLOO,
			},
		},
	}

	trainFile2 := "./testdata/vl/linear_boston_housing/train_dataB.csv"
	samplesFile2, err := ioutil.ReadFile(trainFile2)
	checkErr(err, t)
	req2 := &pbCom.StartTaskRequest{
		TaskID: "TestEvaRegMpc",
		File:   samplesFile2,
		Hosts:  []string{":8080"},
		Params: &pbCom.TaskParams{
			Algo:     pbCom.Algorithm_LINEAR_REGRESSION_VL,
			TaskType: pbCom.TaskType_LEARN,
			TrainParams: &pbCom.TrainParams{
				Label:     "MEDV",
				RegMode:   0,
				RegParam:  0.1,
				Alpha:     0.1,
				Amplitude: 0.0001,
				Accuracy:  10,
				IsTagPart: true,
				IdName:    "id",
				BatchSize: 4,
			},
			EvalParams: &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule_ErLOO,
			},
		},
	}

	// start test
	go mpc1.run()
	go mpc2.run()
	t.Log("Run Mpcs and wait a moment")
	time.Sleep(1 * time.Second)

	go func() {
		t.Log("Mpc1.StartTask")
		err = mpc1.StartTask(req1)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log("Mpc1.StartTask successfully")
	}()

	go func() {
		t.Log("Mpc2.StartTask")
		err = mpc2.StartTask(req2)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log("Mpc2.StartTask successfully")
	}()

	time.Sleep(2 * time.Second)

	// train and evaluate
	var done = make(chan int)
	var stop = make(chan int, 1)
	var stopped1 bool
	var stopped2 bool

	isDone := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	for {
		select {
		case reqTRecv2 := <-reqTC1:
			respTSend2, err := mpc2.Train(reqTRecv2)
			if err != nil {
				t.Logf("Mpc2.Train err: %s", err.Error())
			}
			respTC1 <- respTSend2
		case trainResult1 := <-trainResultC1:
			if trainResult1 == nil { //failed
				t.Logf("mpc1 train model failed")
			} else {
				model1, _ := vl_common.TrainModelsFromBytes(trainResult1.Model)
				t.Logf("mpc1 train and evaluate model successfully, and model is[%v], and evaluation result is[%v]", model1, trainResult1.EvalMetricScores)
			}
			stopped1 = true
			if stopped1 && stopped2 {
				stop <- 1
			}

		case reqTRecv1 := <-reqTC2:
			respTSend1, err := mpc1.Train(reqTRecv1)
			if err != nil {
				t.Logf("Mpc1.Train err: %s", err.Error())
			}
			respTC2 <- respTSend1
		case trainResult2 := <-trainResultC2:
			if trainResult2 == nil { //failed
				t.Logf("mpc2 train model failed")
			} else {
				model2, _ := vl_common.TrainModelsFromBytes(trainResult2.Model)
				t.Logf("mpc2 train and evaluate model successfully, and model is[%v], and evaluation result is[%v]", model2, trainResult2.EvalMetricScores)
			}
			stopped2 = true
			if stopped1 && stopped2 {
				stop <- 1
			}
		case reqPRecv2 := <-reqPC1:
			respPSend2, err := mpc2.Predict(reqPRecv2)
			if err != nil {
				t.Logf("Mpc2.Predict err: %s", err.Error())
			}
			respPC1 <- respPSend2
		case <-predictResultC1:
			t.Error("Something wrong happened, because mpc1 shouldn't return prediction result.")
			t.FailNow()

		case reqPRecv1 := <-reqPC2:
			respPSend1, err := mpc1.Predict(reqPRecv1)
			if err != nil {
				t.Logf("Mpc1.Predict err: %s", err.Error())
			}
			respPC2 <- respPSend1
		case <-predictResultC2:
			t.Error("Something wrong happened, because mpc2 shouldn't return prediction result.")
			t.FailNow()

		case <-stop:
			close(done)
		}

		if isDone() {
			break
		}
	}
} */

func TestEvaluBinClassRandomSplit(t *testing.T) {
	//initiate mpc instance for party1
	var reqTC1 = make(chan *pb.TrainRequest)
	var respTC1 = make(chan *pb.TrainResponse)
	var reqPC1 = make(chan *pb.PredictRequest)
	var respPC1 = make(chan *pb.PredictResponse)
	var trainResultC1 = make(chan *pbCom.TrainTaskResult)
	var predictResultC1 = make(chan *pbCom.PredictTaskResult)

	rpc1 := &rpc{
		reqTrainC:  reqTC1,
		respTrainC: respTC1,

		reqPredC:  reqPC1,
		respPredC: respPC1,

		t: t,
	}

	mh1 := &modelHolder{
		trainResultC:   trainResultC1,
		predictResultC: predictResultC1,
		t:              t,
	}

	mpc1 = &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}

	conf1 := Config{
		Address:          ":8080",
		TrainTaskLimit:   1000,
		PredictTaskLimit: 2000,
		RpcTimeout:       time.Duration(3),
	}

	trainCallback1 := TrainCallBack{ModelHolder: mh1, Mpc: mpc1}
	mpc1.trainer = trainer.NewTrainer(conf1.Address, rpc1, &trainCallback1, conf1.TrainTaskLimit)

	predictCallBack1 := PredictCallBack{ModelHolder: mh1, Mpc: mpc1}
	mpc1.predictor = predictor.NewPredictor(conf1.Address, rpc1, &predictCallBack1, conf1.PredictTaskLimit)

	//initiate mpc instance for party2
	var reqTC2 = make(chan *pb.TrainRequest)
	var respTC2 = make(chan *pb.TrainResponse)
	var reqPC2 = make(chan *pb.PredictRequest)
	var respPC2 = make(chan *pb.PredictResponse)
	var trainResultC2 = make(chan *pbCom.TrainTaskResult)
	var predictResultC2 = make(chan *pbCom.PredictTaskResult)

	rpc2 := &rpc{
		reqTrainC:  reqTC2,
		respTrainC: respTC2,

		reqPredC:  reqPC2,
		respPredC: respPC2,

		t: t,
	}

	mh2 := &modelHolder{
		trainResultC:   trainResultC2,
		predictResultC: predictResultC2,
		t:              t,
	}

	mpc2 = &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}

	conf2 := Config{
		Address:          ":8081",
		TrainTaskLimit:   1000,
		PredictTaskLimit: 2000,
		RpcTimeout:       time.Duration(3),
	}

	trainCallback2 := TrainCallBack{ModelHolder: mh2, Mpc: mpc2}
	mpc2.trainer = trainer.NewTrainer(conf2.Address, rpc2, &trainCallback2, conf2.TrainTaskLimit)

	predictCallBack2 := PredictCallBack{ModelHolder: mh2, Mpc: mpc2}
	mpc2.predictor = predictor.NewPredictor(conf2.Address, rpc2, &predictCallBack2, conf2.PredictTaskLimit)

	// prepare requests
	trainFile1 := "./testdata/vl/logic_iris_plants/dataA.csv"

	samplesFile1, err := ioutil.ReadFile(trainFile1)
	checkErr(err, t)
	req1 := &pbCom.StartTaskRequest{
		TaskID: "TestEvaBinClassMpc",
		File:   samplesFile1,
		Hosts:  []string{":8081"},
		Params: &pbCom.TaskParams{
			Algo:     pbCom.Algorithm_LOGIC_REGRESSION_VL,
			TaskType: pbCom.TaskType_LEARN,
			TrainParams: &pbCom.TrainParams{
				Label:     "Label",
				LabelName: "Iris-setosa",
				RegMode:   0,
				RegParam:  0.1,
				Alpha:     0.1,
				Amplitude: 0.0001,
				Accuracy:  10,
				IsTagPart: false,
				IdName:    "id",
				BatchSize: 4,
			},
			EvalParams: &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule_ErRandomSplit,
				RandomSplit: &pbCom.RandomSplit{
					PercentLO: 10,
				},
			},
		},
	}

	trainFile2 := "./testdata/vl/logic_iris_plants/dataB.csv"
	samplesFile2, err := ioutil.ReadFile(trainFile2)
	checkErr(err, t)
	req2 := &pbCom.StartTaskRequest{
		TaskID: "TestEvaBinClassMpc",
		File:   samplesFile2,
		Hosts:  []string{":8080"},
		Params: &pbCom.TaskParams{
			Algo:     pbCom.Algorithm_LOGIC_REGRESSION_VL,
			TaskType: pbCom.TaskType_LEARN,
			TrainParams: &pbCom.TrainParams{
				Label:     "Label",
				LabelName: "Iris-setosa",
				RegMode:   0,
				RegParam:  0.1,
				Alpha:     0.1,
				Amplitude: 0.0001,
				Accuracy:  10,
				IsTagPart: true,
				IdName:    "id",
				BatchSize: 4,
			},
			EvalParams: &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule_ErRandomSplit,
				RandomSplit: &pbCom.RandomSplit{
					PercentLO: 10,
				},
			},
		},
	}

	// start test
	go mpc1.run()
	go mpc2.run()
	t.Log("Run Mpcs and wait a moment")
	time.Sleep(1 * time.Second)

	go func() {
		t.Log("Mpc1.StartTask")
		err = mpc1.StartTask(req1)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log("Mpc1.StartTask successfully")
	}()

	go func() {
		t.Log("Mpc2.StartTask")
		err = mpc2.StartTask(req2)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log("Mpc2.StartTask successfully")
	}()

	time.Sleep(2 * time.Second)

	// train and evaluate
	var done = make(chan int)
	var stop = make(chan int, 1)
	var stopped1 bool
	var stopped2 bool

	isDone := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	for {
		select {
		case reqTRecv2 := <-reqTC1:
			respTSend2, err := mpc2.Train(reqTRecv2)
			if err != nil {
				t.Logf("Mpc2.Train err: %s", err.Error())
			}
			respTC1 <- respTSend2
		case trainResult1 := <-trainResultC1:
			if trainResult1 == nil { //failed
				t.Logf("mpc1 train model failed")
			} else {
				model1, _ := vl_common.TrainModelsFromBytes(trainResult1.Model)
				t.Logf("mpc1 train and evaluate model successfully, and model is[%v], and evaluation result is[%v]", model1, trainResult1.EvalMetricScores)
			}
			stopped1 = true
			if stopped1 && stopped2 {
				stop <- 1
			}

		case reqTRecv1 := <-reqTC2:
			respTSend1, err := mpc1.Train(reqTRecv1)
			if err != nil {
				t.Logf("Mpc1.Train err: %s", err.Error())
			}
			respTC2 <- respTSend1
		case trainResult2 := <-trainResultC2:
			if trainResult2 == nil { //failed
				t.Logf("mpc2 train model failed")
			} else {
				model2, _ := vl_common.TrainModelsFromBytes(trainResult2.Model)
				t.Logf("mpc2 train and evaluate model successfully, and model is[%v], and evaluation result is[%v]", model2, trainResult2.EvalMetricScores)
			}
			stopped2 = true
			if stopped1 && stopped2 {
				stop <- 1
			}
		case reqPRecv2 := <-reqPC1:
			respPSend2, err := mpc2.Predict(reqPRecv2)
			if err != nil {
				t.Logf("Mpc2.Predict err: %s", err.Error())
			}
			respPC1 <- respPSend2
		case <-predictResultC1:
			t.Error("Something wrong happened, because mpc1 shouldn't return prediction result.")
			t.FailNow()

		case reqPRecv1 := <-reqPC2:
			respPSend1, err := mpc1.Predict(reqPRecv1)
			if err != nil {
				t.Logf("Mpc1.Predict err: %s", err.Error())
			}
			respPC2 <- respPSend1
		case <-predictResultC2:
			t.Error("Something wrong happened, because mpc2 shouldn't return prediction result.")
			t.FailNow()

		case <-stop:
			close(done)
		}

		if isDone() {
			break
		}
	}
}

func TestEvaluBinClassCV(t *testing.T) {
	//initiate mpc instance for party1
	var reqTC1 = make(chan *pb.TrainRequest)
	var respTC1 = make(chan *pb.TrainResponse)
	var reqPC1 = make(chan *pb.PredictRequest)
	var respPC1 = make(chan *pb.PredictResponse)
	var trainResultC1 = make(chan *pbCom.TrainTaskResult)
	var predictResultC1 = make(chan *pbCom.PredictTaskResult)

	rpc1 := &rpc{
		reqTrainC:  reqTC1,
		respTrainC: respTC1,

		reqPredC:  reqPC1,
		respPredC: respPC1,

		t: t,
	}

	mh1 := &modelHolder{
		trainResultC:   trainResultC1,
		predictResultC: predictResultC1,
		t:              t,
	}

	mpc1 = &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}

	conf1 := Config{
		Address:          ":8080",
		TrainTaskLimit:   1000,
		PredictTaskLimit: 2000,
		RpcTimeout:       time.Duration(3),
	}

	trainCallback1 := TrainCallBack{ModelHolder: mh1, Mpc: mpc1}
	mpc1.trainer = trainer.NewTrainer(conf1.Address, rpc1, &trainCallback1, conf1.TrainTaskLimit)

	predictCallBack1 := PredictCallBack{ModelHolder: mh1, Mpc: mpc1}
	mpc1.predictor = predictor.NewPredictor(conf1.Address, rpc1, &predictCallBack1, conf1.PredictTaskLimit)

	//initiate mpc instance for party2
	var reqTC2 = make(chan *pb.TrainRequest)
	var respTC2 = make(chan *pb.TrainResponse)
	var reqPC2 = make(chan *pb.PredictRequest)
	var respPC2 = make(chan *pb.PredictResponse)
	var trainResultC2 = make(chan *pbCom.TrainTaskResult)
	var predictResultC2 = make(chan *pbCom.PredictTaskResult)

	rpc2 := &rpc{
		reqTrainC:  reqTC2,
		respTrainC: respTC2,

		reqPredC:  reqPC2,
		respPredC: respPC2,

		t: t,
	}

	mh2 := &modelHolder{
		trainResultC:   trainResultC2,
		predictResultC: predictResultC2,
		t:              t,
	}

	mpc2 = &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}

	conf2 := Config{
		Address:          ":8081",
		TrainTaskLimit:   1000,
		PredictTaskLimit: 2000,
		RpcTimeout:       time.Duration(3),
	}

	trainCallback2 := TrainCallBack{ModelHolder: mh2, Mpc: mpc2}
	mpc2.trainer = trainer.NewTrainer(conf2.Address, rpc2, &trainCallback2, conf2.TrainTaskLimit)

	predictCallBack2 := PredictCallBack{ModelHolder: mh2, Mpc: mpc2}
	mpc2.predictor = predictor.NewPredictor(conf2.Address, rpc2, &predictCallBack2, conf2.PredictTaskLimit)

	// prepare requests
	trainFile1 := "./testdata/vl/logic_iris_plants/dataA.csv"

	samplesFile1, err := ioutil.ReadFile(trainFile1)
	checkErr(err, t)
	req1 := &pbCom.StartTaskRequest{
		TaskID: "TestEvaBinClassMpc",
		File:   samplesFile1,
		Hosts:  []string{":8081"},
		Params: &pbCom.TaskParams{
			Algo:     pbCom.Algorithm_LOGIC_REGRESSION_VL,
			TaskType: pbCom.TaskType_LEARN,
			TrainParams: &pbCom.TrainParams{
				Label:     "Label",
				LabelName: "Iris-setosa",
				RegMode:   0,
				RegParam:  0.1,
				Alpha:     0.1,
				Amplitude: 0.0001,
				Accuracy:  10,
				IsTagPart: false,
				IdName:    "id",
				BatchSize: 4,
			},
			EvalParams: &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule_ErCrossVal,
				Cv: &pbCom.CrossVal{
					Folds:   10,
					Shuffle: true,
				},
			},
		},
	}

	trainFile2 := "./testdata/vl/logic_iris_plants/dataB.csv"
	samplesFile2, err := ioutil.ReadFile(trainFile2)
	checkErr(err, t)
	req2 := &pbCom.StartTaskRequest{
		TaskID: "TestEvaBinClassMpc",
		File:   samplesFile2,
		Hosts:  []string{":8080"},
		Params: &pbCom.TaskParams{
			Algo:     pbCom.Algorithm_LOGIC_REGRESSION_VL,
			TaskType: pbCom.TaskType_LEARN,
			TrainParams: &pbCom.TrainParams{
				Label:     "Label",
				LabelName: "Iris-setosa",
				RegMode:   0,
				RegParam:  0.1,
				Alpha:     0.1,
				Amplitude: 0.0001,
				Accuracy:  10,
				IsTagPart: true,
				IdName:    "id",
				BatchSize: 4,
			},
			EvalParams: &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule_ErCrossVal,
				Cv: &pbCom.CrossVal{
					Folds:   10,
					Shuffle: true,
				},
			},
		},
	}

	// start test
	go mpc1.run()
	go mpc2.run()
	t.Log("Run Mpcs and wait a moment")
	time.Sleep(1 * time.Second)

	go func() {
		t.Log("Mpc1.StartTask")
		err = mpc1.StartTask(req1)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log("Mpc1.StartTask successfully")
	}()

	go func() {
		t.Log("Mpc2.StartTask")
		err = mpc2.StartTask(req2)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log("Mpc2.StartTask successfully")
	}()

	time.Sleep(2 * time.Second)

	// train and evaluate
	var done = make(chan int)
	var stop = make(chan int, 1)
	var stopped1 bool
	var stopped2 bool

	isDone := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	for {
		select {
		case reqTRecv2 := <-reqTC1:
			respTSend2, err := mpc2.Train(reqTRecv2)
			if err != nil {
				t.Logf("Mpc2.Train err: %s", err.Error())
			}
			respTC1 <- respTSend2
		case trainResult1 := <-trainResultC1:
			if trainResult1 == nil { //failed
				t.Logf("mpc1 train model failed")
			} else {
				model1, _ := vl_common.TrainModelsFromBytes(trainResult1.Model)
				t.Logf("mpc1 train and evaluate model successfully, and model is[%v], and evaluation result is[%v]", model1, trainResult1.EvalMetricScores)
			}
			stopped1 = true
			if stopped1 && stopped2 {
				stop <- 1
			}

		case reqTRecv1 := <-reqTC2:
			respTSend1, err := mpc1.Train(reqTRecv1)
			if err != nil {
				t.Logf("Mpc1.Train err: %s", err.Error())
			}
			respTC2 <- respTSend1
		case trainResult2 := <-trainResultC2:
			if trainResult2 == nil { //failed
				t.Logf("mpc2 train model failed")
			} else {
				model2, _ := vl_common.TrainModelsFromBytes(trainResult2.Model)
				t.Logf("mpc2 train and evaluate model successfully, and model is[%v], and evaluation result is[%v]", model2, trainResult2.EvalMetricScores)
			}
			stopped2 = true
			if stopped1 && stopped2 {
				stop <- 1
			}
		case reqPRecv2 := <-reqPC1:
			respPSend2, err := mpc2.Predict(reqPRecv2)
			if err != nil {
				t.Logf("Mpc2.Predict err: %s", err.Error())
			}
			respPC1 <- respPSend2
		case <-predictResultC1:
			t.Error("Something wrong happened, because mpc1 shouldn't return prediction result.")
			t.FailNow()

		case reqPRecv1 := <-reqPC2:
			respPSend1, err := mpc1.Predict(reqPRecv1)
			if err != nil {
				t.Logf("Mpc1.Predict err: %s", err.Error())
			}
			respPC2 <- respPSend1
		case <-predictResultC2:
			t.Error("Something wrong happened, because mpc2 shouldn't return prediction result.")
			t.FailNow()

		case <-stop:
			close(done)
		}

		if isDone() {
			break
		}
	}
}

func TestEvaluBinClassLoo(t *testing.T) {
	//initiate mpc instance for party1
	var reqTC1 = make(chan *pb.TrainRequest)
	var respTC1 = make(chan *pb.TrainResponse)
	var reqPC1 = make(chan *pb.PredictRequest)
	var respPC1 = make(chan *pb.PredictResponse)
	var trainResultC1 = make(chan *pbCom.TrainTaskResult)
	var predictResultC1 = make(chan *pbCom.PredictTaskResult)

	rpc1 := &rpc{
		reqTrainC:  reqTC1,
		respTrainC: respTC1,

		reqPredC:  reqPC1,
		respPredC: respPC1,

		t: t,
	}

	mh1 := &modelHolder{
		trainResultC:   trainResultC1,
		predictResultC: predictResultC1,
		t:              t,
	}

	mpc1 = &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}

	conf1 := Config{
		Address:          ":8080",
		TrainTaskLimit:   1000,
		PredictTaskLimit: 2000,
		RpcTimeout:       time.Duration(3),
	}

	trainCallback1 := TrainCallBack{ModelHolder: mh1, Mpc: mpc1}
	mpc1.trainer = trainer.NewTrainer(conf1.Address, rpc1, &trainCallback1, conf1.TrainTaskLimit)

	predictCallBack1 := PredictCallBack{ModelHolder: mh1, Mpc: mpc1}
	mpc1.predictor = predictor.NewPredictor(conf1.Address, rpc1, &predictCallBack1, conf1.PredictTaskLimit)

	//initiate mpc instance for party2
	var reqTC2 = make(chan *pb.TrainRequest)
	var respTC2 = make(chan *pb.TrainResponse)
	var reqPC2 = make(chan *pb.PredictRequest)
	var respPC2 = make(chan *pb.PredictResponse)
	var trainResultC2 = make(chan *pbCom.TrainTaskResult)
	var predictResultC2 = make(chan *pbCom.PredictTaskResult)

	rpc2 := &rpc{
		reqTrainC:  reqTC2,
		respTrainC: respTC2,

		reqPredC:  reqPC2,
		respPredC: respPC2,

		t: t,
	}

	mh2 := &modelHolder{
		trainResultC:   trainResultC2,
		predictResultC: predictResultC2,
		t:              t,
	}

	mpc2 = &mpc{
		stopC:    make(chan struct{}),
		doneC:    make(chan struct{}),
		trainC:   make(chan trainRequest),
		predictC: make(chan predictRequest),
	}

	conf2 := Config{
		Address:          ":8081",
		TrainTaskLimit:   1000,
		PredictTaskLimit: 2000,
		RpcTimeout:       time.Duration(3),
	}

	trainCallback2 := TrainCallBack{ModelHolder: mh2, Mpc: mpc2}
	mpc2.trainer = trainer.NewTrainer(conf2.Address, rpc2, &trainCallback2, conf2.TrainTaskLimit)

	predictCallBack2 := PredictCallBack{ModelHolder: mh2, Mpc: mpc2}
	mpc2.predictor = predictor.NewPredictor(conf2.Address, rpc2, &predictCallBack2, conf2.PredictTaskLimit)

	// prepare requests
	trainFile1 := "./testdata/vl/logic_iris_plants/dataA.csv"

	samplesFile1, err := ioutil.ReadFile(trainFile1)
	checkErr(err, t)
	req1 := &pbCom.StartTaskRequest{
		TaskID: "TestEvaBinClassMpc",
		File:   samplesFile1,
		Hosts:  []string{":8081"},
		Params: &pbCom.TaskParams{
			Algo:     pbCom.Algorithm_LOGIC_REGRESSION_VL,
			TaskType: pbCom.TaskType_LEARN,
			TrainParams: &pbCom.TrainParams{
				Label:     "Label",
				LabelName: "Iris-setosa",
				RegMode:   0,
				RegParam:  0.1,
				Alpha:     0.1,
				Amplitude: 0.0001,
				Accuracy:  10,
				IsTagPart: false,
				IdName:    "id",
				BatchSize: 4,
			},
			EvalParams: &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule_ErLOO,
			},
		},
	}

	trainFile2 := "./testdata/vl/logic_iris_plants/dataB.csv"
	samplesFile2, err := ioutil.ReadFile(trainFile2)
	checkErr(err, t)
	req2 := &pbCom.StartTaskRequest{
		TaskID: "TestEvaBinClassMpc",
		File:   samplesFile2,
		Hosts:  []string{":8080"},
		Params: &pbCom.TaskParams{
			Algo:     pbCom.Algorithm_LOGIC_REGRESSION_VL,
			TaskType: pbCom.TaskType_LEARN,
			TrainParams: &pbCom.TrainParams{
				Label:     "Label",
				LabelName: "Iris-setosa",
				RegMode:   0,
				RegParam:  0.1,
				Alpha:     0.1,
				Amplitude: 0.0001,
				Accuracy:  10,
				IsTagPart: true,
				IdName:    "id",
				BatchSize: 4,
			},
			EvalParams: &pbCom.EvaluationParams{
				Enable:   true,
				EvalRule: pbCom.EvaluationRule_ErLOO,
			},
		},
	}

	// start test
	go mpc1.run()
	go mpc2.run()
	t.Log("Run Mpcs and wait a moment")
	time.Sleep(1 * time.Second)

	go func() {
		t.Log("Mpc1.StartTask")
		err = mpc1.StartTask(req1)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log("Mpc1.StartTask successfully")
	}()

	go func() {
		t.Log("Mpc2.StartTask")
		err = mpc2.StartTask(req2)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		t.Log("Mpc2.StartTask successfully")
	}()

	time.Sleep(2 * time.Second)

	// train and evaluate
	var done = make(chan int)
	var stop = make(chan int, 1)
	var stopped1 bool
	var stopped2 bool

	isDone := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	for {
		select {
		case reqTRecv2 := <-reqTC1:
			respTSend2, err := mpc2.Train(reqTRecv2)
			if err != nil {
				t.Logf("Mpc2.Train err: %s", err.Error())
			}
			respTC1 <- respTSend2
		case trainResult1 := <-trainResultC1:
			if trainResult1 == nil { //failed
				t.Logf("mpc1 train model failed")
			} else {
				model1, _ := vl_common.TrainModelsFromBytes(trainResult1.Model)
				t.Logf("mpc1 train and evaluate model successfully, and model is[%v], and evaluation result is[%v]", model1, trainResult1.EvalMetricScores)
			}
			stopped1 = true
			if stopped1 && stopped2 {
				stop <- 1
			}

		case reqTRecv1 := <-reqTC2:
			respTSend1, err := mpc1.Train(reqTRecv1)
			if err != nil {
				t.Logf("Mpc1.Train err: %s", err.Error())
			}
			respTC2 <- respTSend1
		case trainResult2 := <-trainResultC2:
			if trainResult2 == nil { //failed
				t.Logf("mpc2 train model failed")
			} else {
				model2, _ := vl_common.TrainModelsFromBytes(trainResult2.Model)
				t.Logf("mpc2 train and evaluate model successfully, and model is[%v], and evaluation result is[%v]", model2, trainResult2.EvalMetricScores)
			}
			stopped2 = true
			if stopped1 && stopped2 {
				stop <- 1
			}
		case reqPRecv2 := <-reqPC1:
			respPSend2, err := mpc2.Predict(reqPRecv2)
			if err != nil {
				t.Logf("Mpc2.Predict err: %s", err.Error())
			}
			respPC1 <- respPSend2
		case <-predictResultC1:
			t.Error("Something wrong happened, because mpc1 shouldn't return prediction result.")
			t.FailNow()

		case reqPRecv1 := <-reqPC2:
			respPSend1, err := mpc1.Predict(reqPRecv1)
			if err != nil {
				t.Logf("Mpc1.Predict err: %s", err.Error())
			}
			respPC2 <- respPSend1
		case <-predictResultC2:
			t.Error("Something wrong happened, because mpc2 shouldn't return prediction result.")
			t.FailNow()

		case <-stop:
			close(done)
		}

		if isDone() {
			break
		}
	}
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}
