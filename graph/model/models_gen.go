// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

type CreateJobArgs struct {
	Expr    string `json:"expr"`
	Name    string `json:"name"`
	State   string `json:"state"`
	Timeout *int   `json:"timeout"`
}

type QueryJobsArgs struct {
	Limit  int    `json:"limit"`
	Skip   int    `json:"skip"`
	Filter string `json:"filter"`
}

type UpdateJobArgs struct {
	Expr    *string `json:"expr"`
	State   *string `json:"state"`
	Timeout *int    `json:"timeout"`
}
