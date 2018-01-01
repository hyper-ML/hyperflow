package main_test

import (
  "os"
  "strings"
  "fmt"
  "time"
  "testing"
  "net/url"
  "io/ioutil"
  "bytes"
  "encoding/json"
  flow "hyperview.in/server/core/flow"
  ws "hyperview.in/server/core/workspace"
  client "hyperview.in/server/test_utils/rest_client"
)

// create a rest client 
// call the server to create a new repo
// start a commit 
// send a file to server to commit 
// execute a flow 
// monitor output 
// review output 

const (
  SERVER_ADDR = "http://localhost:8888"
  API_BASE_PATH = "/"
  TEST_REPO_NAME = "test_repo0123456789"
  TEST_BRANCH_NAME ="master"
  TEST_TEMP_DIR = "/var/tmp"
  TEST_PY_FILE_NAME = "p.py"
  TEST_CMD_STRING = "python p.py"
)

func getRestClient(api_path string) client.Interface {
  base_url, _ := url.Parse(SERVER_ADDR+ API_BASE_PATH)
  client, err := client.NewRESTClient(base_url, api_path, nil)
  if err != nil {
    fmt.Println("Failed to create NewRESTClient: ", err)
  }
  return client
}

func createRepo(repoName, branchName string) (*ws.Repo, *ws.Branch, error) {
  client := getRestClient("repo")
  repo_req := client.Verb("POST")
  repo_req.Param("repoName", repoName)
  repo_req.Param("branchName", branchName)
  resp := repo_req.Do()
  _, err := resp.Raw()

  if err != nil {
    return nil, nil, fmt.Errorf("Failed while initializing repo: %s", err.Error())
  }

  return &ws.Repo{Name: repoName}, &ws.Branch{Name: "master"}, nil
}


func getRepo(repoName, branchName string) (*ws.Repo, *ws.Branch, error) {
  
  client := getRestClient("repo_attrs")
  repo_req := client.Verb("GET")
  repo_req.Param("repoName", repoName)
  repo_req.Param("branchName", branchName)
  resp := repo_req.Do()
  json_body, err := resp.Raw()

  if err != nil {
    return nil, nil, fmt.Errorf("Failed while initializing repo: %s", err.Error())
  }

  repo_attrs := ws.RepoAttrs{}
  err = json.Unmarshal(json_body, &repo_attrs)
   
  return repo_attrs.Repo, repo_attrs.Branches[branchName], nil
}

func initCommit(repo *ws.Repo, branch *ws.Branch) (*ws.CommitAttrs, error) {
  client := getRestClient("commit")
  req := client.Verb("GET")
  req.Param("repoName", repo.Name)
  req.Param("branchName", branch.Name)
  req.Param("commitId", "")
  resp := req.Do()
  json_body, err := resp.Raw()

  if err != nil {
    return nil, fmt.Errorf("Failed while initializing commit: %s", err.Error())
  }

  commit_attrs := ws.CommitAttrs{}
  err = json.Unmarshal(json_body, &commit_attrs)

  if err != nil {
    return nil, err
  }

  return &commit_attrs, nil
}


func closeCommit(repo *ws.Repo, branch *ws.Branch, commit *ws.Commit) (error) {
  client := getRestClient("commit")
  req := client.Verb("POST")
  req.Param("repoName", repo.Name)
  req.Param("branchName", branch.Name)
  req.Param("commitId", commit.Id)

  resp := req.Do()
  _, err := resp.Raw()

  if err != nil {
    return fmt.Errorf("Failed while closing commit: %s", err.Error())
  }

  return nil
}

func createDirIfNotExist(dir string) error{
  if _, err := os.Stat(dir); os.IsNotExist(err) {
    err = os.MkdirAll(dir, 0755)
    if err != nil {
      return err
    }
  }
  return nil
}

func genSourceFile(repo_name string) (repoDir string, f *os.File, err error){
  repo_dir := TEST_TEMP_DIR + "/" + strings.Replace(repo_name, "/", "", -1)

  err = createDirIfNotExist(repo_dir)
  if err != nil {
    return "", nil, err
  }
  sep := ""
  if repo_dir[len(repo_dir)-1:] != "/" {
    sep = "/"
  }
  file_path:= repo_dir + sep + TEST_PY_FILE_NAME
  
  f, err = os.Create(file_path)
  source := []byte("print(\"hello\")\n")
  _, err = f.Write(source)

  if err != nil {
    return "", nil, err
  }

  repoDir = repo_dir
  return 
}

func genSource() ([]byte) {
  return []byte("import os \nprint(\"hello\")\nf = open(\"/wh_data/saved_models/model.txt\",\"w+\") \nprint(\"1\") \nf.write(\"Hello World\") \nprint(\"2\") \nf.close() \nprint(\"3\") \nfor z in os.listdir(\"/wh_data/\"):\n  print(z)")
  //return []byte("import os\nprint(\"hello\")\nos.makedirs(\"/home/workspace/pip\")\n")
}

func pushCode(code []byte, fpath string, repoName string, branchName string, commitId string) (error) {
  client := getRestClient("vfs")

  r := client.VerbSp("PUT", "put_file")
  r.Param("repoName", repoName)
  r.Param("branchName", branchName)
  r.Param("commitId", commitId)
  r.Param("path", fpath)
  r.Param("size", string(len(code)))

  _ = r.SetBodyReader(ioutil.NopCloser(bytes.NewReader(code)))

  resp := r.Do()

  if resp.Error()!= nil {
    return resp.Error()
  }

  put_response := ws.PutFileResponse{}
  err := json.Unmarshal(resp.Body(), &put_response)
  if err != nil {
    return err
  }

  fmt.Println("put_response: ", put_response.Written)
  return nil
}

func StartTask(repo *ws.Repo, branch *ws.Branch, commit *ws.Commit, cmdStr string) (*flow.Flow, error) {
  task_req := flow.NewFlowLaunchRequest {
    Repo: *repo,
    Branch: *branch,
    Commit: *commit, 
    CmdString: cmdStr,
  }

  json_str, _ := json.Marshal(&task_req) 
  
  client := getRestClient("flow")
  api_call := client.Verb("POST") 
  _ = api_call.SetBodyReader(ioutil.NopCloser(bytes.NewReader(json_str)))

  resp := api_call.Do()
  api_resp, _ := resp.Raw()  

  task_result :=  flow.NewFlowLaunchResponse{}
  _ = json.Unmarshal(api_resp, &task_result)

  fmt.Println("[RunTask] Flow Id: ", task_result.Flow.Id)

  return task_result.Flow, nil
}

func getLogSize(flow *flow.Flow) (int, error) {
  client := getRestClient("tasks")
  url_subpath := "/" + flow.Id + "/log"

  log_req := client.VerbSp("GET", url_subpath)
  log_resp := log_req.Do()

  log_bytes, err:= log_resp.Raw() 
  log_len:= len(log_bytes)

  fmt.Println("[ApiClient.RequestLog] log_bytes:", log_len, err)  

  return log_len, nil
}

func getSavedModelSize(repo *ws.Repo, branch *ws.Branch, commit *ws.Commit) (int, error) {
  var err error

  client := getRestClient("repo_attrs")
  url_subpath:= "/" + repo.Name + "/branch/" + branch.Name + "/commit/" + commit.Id + "/model_repo" 

  req := client.VerbSp("GET", url_subpath)
  resp := req.Do()

  json_body, err := resp.Raw()
  model_repo := ws.RepoAttrs{}
  err = json.Unmarshal(json_body, &model_repo)
  if err != nil {
    fmt.Println("[getSavedModelSize] Failed to unmarshal model request response ")
    return 0, err
  }

  if model_repo.Repo.Name != "" {
    fmt.Println("Model Repo found: ", model_repo.Repo.Name)
    url_subpath = "/" + repo.Name + "/branch/" + branch.Name + "/commit/" + commit.Id + "/size"
    size_req := client.VerbSp("GET", url_subpath)
    resp := req.Do()
    json_body, err := resp.Raw()
    size_resp := ws.CommitSizeResponse{}
    if err := json.Unmarshal(json_body, &model_repo); err != nil {
      fmt.Println("[getSavedModelSize] Failed to unmarshal size request response ")
      return 0, err
    }
    return size_resp.Size, nil
  }
  return 0, nil 
}

func Test_WorkerCycle(t *testing.T) {
  var repo *ws.Repo
  var branch *ws.Branch
  var err error
  var branch_name string = TEST_BRANCH_NAME
  var repo_name string = TEST_REPO_NAME
  var file_path string = TEST_PY_FILE_NAME
  var cmd_string string = TEST_CMD_STRING

  repo, branch, err= getRepo(repo_name, branch_name)
  fmt.Println("Received repo, branch:", repo, branch)
  
  if err != nil {
    fmt.Println("get_repo_error: ", err)
    t.Fatalf("get_repo_error: %s", err.Error())
  }
  
  if repo == nil || repo.Name == "" {
    fmt.Println("creating test repo... ", repo_name)
    repo, branch, err = createRepo(repo_name, branch_name)
    if err != nil {
      t.Fatalf("create_repo_error: %s", err.Error())
    }
    fmt.Println("Created repo, branch, id", repo, branch)
  } 

  commit_attrs, err := initCommit(repo, branch)
  if err != nil {
    t.Fatalf("init_commit_error: %s", err.Error())
  }
  fmt.Println("Commit Initialized: ", commit_attrs.Commit.Id)

  sample_code := genSource()
  err = pushCode(sample_code, file_path, repo.Name, branch.Name, commit_attrs.Commit.Id)
  if err != nil {
    t.Fatalf("push_code_error: %s", err)
  }

  err = closeCommit(repo, branch, commit_attrs.Commit)
  if err != nil {
    t.Fatalf("close_commit_error: %s", err)
  } 

  flw, err := StartTask(repo, branch, commit_attrs.Commit, cmd_string)
  if err != nil {
    t.Fatalf("start_task_error: %s", err)
  }
  fmt.Println("flow id: ", flw.Id)
  
  /* Check log exists */ 
  //wait for task to finish
  fmt.Println("Wait for task to finish (15s)...")
  time.Sleep(15 * time.Second)
  log_len, err := getLogSize(flw)
  if err != nil {
    t.Fatalf("log_error: %s", err)
  }
  if log_len == 0 {
    fmt.Println("Failed to retrieve the task flow log OR log is empty")
    t.Fail()
  }

  /* Check model file exists */ 
  s, err := getSavedModelSize(repo, branch, commit)
  if err != nil {
    t.Fatalf("[getSavedModelSize] %s", err.Error())
  }
  if s == 0 {
    fmt.Println("Failed to retrieve model or found empty file")
    t.Fail()
  }
  
}




