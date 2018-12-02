package main

import (
	"context"
	"fmt"

	"github.com/Shivam010/pg-trigger-logs"
)

func main() {
	dbconfig := "host=localhost port=5432 user=postgres password=password dbname=dtest sslmode=disable"
	ls, err := pgtl.SetupEverything(context.Background(), dbconfig)
	if err != nil {
		panic(err)
	}
	defer pgtl.Unlisten(ls)
	fmt.Println("Setup Done. Event listening")

	for logRes := range pgtl.GetChangesLogs(ls) {
		if logRes.Err != nil {
			panic(err)
		}
		fmt.Printf("%v\n%s\n\n", logRes.Map, logRes.JSON)
	}
}

/*
CREATE DATABASE dtest;
CREATE SCHEMA stest;
CREATE TABLE ptest (id SERIAL PRIMARY KEY, name text NOT NULL);
CREATE TABLE stest.test (id SERIAL PRIMARY KEY, name text NOT NULL);
*/
