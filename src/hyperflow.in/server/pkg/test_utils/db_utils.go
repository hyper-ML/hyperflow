package test_utils

import (
  "hyperflow.in/server/pkg/db"

)
const (
  test_db_user = "apple"
  test_db_password = ""
  test_db_name = "amp_db"
  test_repo_name = "test_repo"
  test_dir = "test_dir"
) 


func FakeDb() (*db.DatabaseContext, error) {
  conn, err := db.NewPostgresContext(test_db_name, test_db_user, test_db_password)
  return conn, err
}