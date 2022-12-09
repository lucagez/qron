import { GraphQLClient } from 'graphql-request';
import * as Dom from 'graphql-request/dist/types.dom';
import gql from 'graphql-tag';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: string;
  String: string;
  Boolean: boolean;
  Int: number;
  Float: number;
  Time: any;
};

export type CreateJobArgs = {
  expr: Scalars['String'];
  meta?: InputMaybe<Scalars['String']>;
  name: Scalars['String'];
  start_at?: InputMaybe<Scalars['Time']>;
  state: Scalars['String'];
  timeout?: InputMaybe<Scalars['Int']>;
};

export type Mutation = {
  __typename?: 'Mutation';
  commitJobs: Array<Scalars['ID']>;
  createJob: TinyJob;
  deleteJobByID: TinyJob;
  deleteJobByName: TinyJob;
  failJobs: Array<Scalars['ID']>;
  fetchForProcessing: Array<TinyJob>;
  retryJobs: Array<Scalars['ID']>;
  updateJobById: TinyJob;
  updateJobByName: TinyJob;
};


export type MutationCommitJobsArgs = {
  ids: Array<Scalars['ID']>;
};


export type MutationCreateJobArgs = {
  args: CreateJobArgs;
};


export type MutationDeleteJobByIdArgs = {
  id: Scalars['ID'];
};


export type MutationDeleteJobByNameArgs = {
  name: Scalars['String'];
};


export type MutationFailJobsArgs = {
  ids: Array<Scalars['ID']>;
};


export type MutationFetchForProcessingArgs = {
  limit?: Scalars['Int'];
};


export type MutationRetryJobsArgs = {
  ids: Array<Scalars['ID']>;
};


export type MutationUpdateJobByIdArgs = {
  args: UpdateJobArgs;
  id: Scalars['ID'];
};


export type MutationUpdateJobByNameArgs = {
  args: UpdateJobArgs;
  name: Scalars['String'];
};

export type Query = {
  __typename?: 'Query';
  queryJobByID: TinyJob;
  queryJobByName: TinyJob;
  searchJobs: Array<TinyJob>;
};


export type QueryQueryJobByIdArgs = {
  id: Scalars['ID'];
};


export type QueryQueryJobByNameArgs = {
  name: Scalars['String'];
};


export type QuerySearchJobsArgs = {
  args: QueryJobsArgs;
};

export type QueryJobsArgs = {
  filter: Scalars['String'];
  limit?: Scalars['Int'];
  skip?: Scalars['Int'];
};

export type TinyJob = {
  __typename?: 'TinyJob';
  created_at: Scalars['Time'];
  executor: Scalars['String'];
  expr: Scalars['String'];
  id: Scalars['ID'];
  last_run_at?: Maybe<Scalars['Time']>;
  meta: Scalars['String'];
  name?: Maybe<Scalars['String']>;
  run_at: Scalars['Time'];
  start_at?: Maybe<Scalars['Time']>;
  state?: Maybe<Scalars['String']>;
  status: Scalars['String'];
  timeout?: Maybe<Scalars['Int']>;
};

export type UpdateJobArgs = {
  expr?: InputMaybe<Scalars['String']>;
  state?: InputMaybe<Scalars['String']>;
  timeout?: InputMaybe<Scalars['Int']>;
};

export type TinyPropsFragment = { __typename?: 'TinyJob', id: string, name?: string | null, expr: string, run_at: any, last_run_at?: any | null, timeout?: number | null, created_at: any, executor: string, state?: string | null, status: string, meta: string };

export type SearchJobsQueryVariables = Exact<{
  args: QueryJobsArgs;
}>;


export type SearchJobsQuery = { __typename?: 'Query', searchJobs: Array<{ __typename?: 'TinyJob', id: string, name?: string | null, expr: string, run_at: any, last_run_at?: any | null, timeout?: number | null, created_at: any, executor: string, state?: string | null, status: string, meta: string }> };

export type QueryJobByNameQueryVariables = Exact<{
  name: Scalars['String'];
}>;


export type QueryJobByNameQuery = { __typename?: 'Query', queryJobByName: { __typename?: 'TinyJob', id: string, name?: string | null, expr: string, run_at: any, last_run_at?: any | null, timeout?: number | null, created_at: any, executor: string, state?: string | null, status: string, meta: string } };

export type QueryJobByIdQueryVariables = Exact<{
  id: Scalars['ID'];
}>;


export type QueryJobByIdQuery = { __typename?: 'Query', queryJobByID: { __typename?: 'TinyJob', id: string, name?: string | null, expr: string, run_at: any, last_run_at?: any | null, timeout?: number | null, created_at: any, executor: string, state?: string | null, status: string, meta: string } };

export type CreateJobMutationVariables = Exact<{
  args: CreateJobArgs;
}>;


export type CreateJobMutation = { __typename?: 'Mutation', createJob: { __typename?: 'TinyJob', id: string, name?: string | null, expr: string, run_at: any, last_run_at?: any | null, timeout?: number | null, created_at: any, executor: string, state?: string | null, status: string, meta: string } };

export type UpdateJobByNameMutationVariables = Exact<{
  name: Scalars['String'];
  args: UpdateJobArgs;
}>;


export type UpdateJobByNameMutation = { __typename?: 'Mutation', updateJobByName: { __typename?: 'TinyJob', id: string, name?: string | null, expr: string, run_at: any, last_run_at?: any | null, timeout?: number | null, created_at: any, executor: string, state?: string | null, status: string, meta: string } };

export type UpdateJobByIdMutationVariables = Exact<{
  id: Scalars['ID'];
  args: UpdateJobArgs;
}>;


export type UpdateJobByIdMutation = { __typename?: 'Mutation', updateJobById: { __typename?: 'TinyJob', id: string, name?: string | null, expr: string, run_at: any, last_run_at?: any | null, timeout?: number | null, created_at: any, executor: string, state?: string | null, status: string, meta: string } };

export type DeleteJobByNameMutationVariables = Exact<{
  name: Scalars['String'];
}>;


export type DeleteJobByNameMutation = { __typename?: 'Mutation', deleteJobByName: { __typename?: 'TinyJob', id: string, name?: string | null, expr: string, run_at: any, last_run_at?: any | null, timeout?: number | null, created_at: any, executor: string, state?: string | null, status: string, meta: string } };

export type DeleteJobByIdMutationVariables = Exact<{
  id: Scalars['ID'];
}>;


export type DeleteJobByIdMutation = { __typename?: 'Mutation', deleteJobByID: { __typename?: 'TinyJob', id: string, name?: string | null, expr: string, run_at: any, last_run_at?: any | null, timeout?: number | null, created_at: any, executor: string, state?: string | null, status: string, meta: string } };

export type FetchForProcessingMutationVariables = Exact<{
  limit?: Scalars['Int'];
}>;


export type FetchForProcessingMutation = { __typename?: 'Mutation', fetchForProcessing: Array<{ __typename?: 'TinyJob', id: string, name?: string | null, expr: string, run_at: any, last_run_at?: any | null, timeout?: number | null, created_at: any, executor: string, state?: string | null, status: string, meta: string }> };

export type CommitJobsMutationVariables = Exact<{
  ids: Array<Scalars['ID']> | Scalars['ID'];
}>;


export type CommitJobsMutation = { __typename?: 'Mutation', commitJobs: Array<string> };

export type FailJobsMutationVariables = Exact<{
  ids: Array<Scalars['ID']> | Scalars['ID'];
}>;


export type FailJobsMutation = { __typename?: 'Mutation', failJobs: Array<string> };

export type RetryJobsMutationVariables = Exact<{
  ids: Array<Scalars['ID']> | Scalars['ID'];
}>;


export type RetryJobsMutation = { __typename?: 'Mutation', retryJobs: Array<string> };

export const TinyPropsFragmentDoc = gql`
    fragment TinyProps on TinyJob {
  id
  name
  expr
  run_at
  last_run_at
  timeout
  created_at
  executor
  state
  status
  meta
}
    `;
export const SearchJobsDocument = gql`
    query searchJobs($args: QueryJobsArgs!) {
  searchJobs(args: $args) {
    ...TinyProps
  }
}
    ${TinyPropsFragmentDoc}`;
export const QueryJobByNameDocument = gql`
    query queryJobByName($name: String!) {
  queryJobByName(name: $name) {
    ...TinyProps
  }
}
    ${TinyPropsFragmentDoc}`;
export const QueryJobByIdDocument = gql`
    query queryJobByID($id: ID!) {
  queryJobByID(id: $id) {
    ...TinyProps
  }
}
    ${TinyPropsFragmentDoc}`;
export const CreateJobDocument = gql`
    mutation createJob($args: CreateJobArgs!) {
  createJob(args: $args) {
    ...TinyProps
  }
}
    ${TinyPropsFragmentDoc}`;
export const UpdateJobByNameDocument = gql`
    mutation updateJobByName($name: String!, $args: UpdateJobArgs!) {
  updateJobByName(name: $name, args: $args) {
    ...TinyProps
  }
}
    ${TinyPropsFragmentDoc}`;
export const UpdateJobByIdDocument = gql`
    mutation updateJobById($id: ID!, $args: UpdateJobArgs!) {
  updateJobById(id: $id, args: $args) {
    ...TinyProps
  }
}
    ${TinyPropsFragmentDoc}`;
export const DeleteJobByNameDocument = gql`
    mutation deleteJobByName($name: String!) {
  deleteJobByName(name: $name) {
    ...TinyProps
  }
}
    ${TinyPropsFragmentDoc}`;
export const DeleteJobByIdDocument = gql`
    mutation deleteJobByID($id: ID!) {
  deleteJobByID(id: $id) {
    ...TinyProps
  }
}
    ${TinyPropsFragmentDoc}`;
export const FetchForProcessingDocument = gql`
    mutation fetchForProcessing($limit: Int! = 50) {
  fetchForProcessing(limit: $limit) {
    ...TinyProps
  }
}
    ${TinyPropsFragmentDoc}`;
export const CommitJobsDocument = gql`
    mutation commitJobs($ids: [ID!]!) {
  commitJobs(ids: $ids)
}
    `;
export const FailJobsDocument = gql`
    mutation failJobs($ids: [ID!]!) {
  failJobs(ids: $ids)
}
    `;
export const RetryJobsDocument = gql`
    mutation retryJobs($ids: [ID!]!) {
  retryJobs(ids: $ids)
}
    `;

export type SdkFunctionWrapper = <T>(action: (requestHeaders?:Record<string, string>) => Promise<T>, operationName: string, operationType?: string) => Promise<T>;


const defaultWrapper: SdkFunctionWrapper = (action, _operationName, _operationType) => action();

export function getSdk(client: GraphQLClient, withWrapper: SdkFunctionWrapper = defaultWrapper) {
  return {
    searchJobs(variables: SearchJobsQueryVariables, requestHeaders?: Dom.RequestInit["headers"]): Promise<SearchJobsQuery> {
      return withWrapper((wrappedRequestHeaders) => client.request<SearchJobsQuery>(SearchJobsDocument, variables, {...requestHeaders, ...wrappedRequestHeaders}), 'searchJobs', 'query');
    },
    queryJobByName(variables: QueryJobByNameQueryVariables, requestHeaders?: Dom.RequestInit["headers"]): Promise<QueryJobByNameQuery> {
      return withWrapper((wrappedRequestHeaders) => client.request<QueryJobByNameQuery>(QueryJobByNameDocument, variables, {...requestHeaders, ...wrappedRequestHeaders}), 'queryJobByName', 'query');
    },
    queryJobByID(variables: QueryJobByIdQueryVariables, requestHeaders?: Dom.RequestInit["headers"]): Promise<QueryJobByIdQuery> {
      return withWrapper((wrappedRequestHeaders) => client.request<QueryJobByIdQuery>(QueryJobByIdDocument, variables, {...requestHeaders, ...wrappedRequestHeaders}), 'queryJobByID', 'query');
    },
    createJob(variables: CreateJobMutationVariables, requestHeaders?: Dom.RequestInit["headers"]): Promise<CreateJobMutation> {
      return withWrapper((wrappedRequestHeaders) => client.request<CreateJobMutation>(CreateJobDocument, variables, {...requestHeaders, ...wrappedRequestHeaders}), 'createJob', 'mutation');
    },
    updateJobByName(variables: UpdateJobByNameMutationVariables, requestHeaders?: Dom.RequestInit["headers"]): Promise<UpdateJobByNameMutation> {
      return withWrapper((wrappedRequestHeaders) => client.request<UpdateJobByNameMutation>(UpdateJobByNameDocument, variables, {...requestHeaders, ...wrappedRequestHeaders}), 'updateJobByName', 'mutation');
    },
    updateJobById(variables: UpdateJobByIdMutationVariables, requestHeaders?: Dom.RequestInit["headers"]): Promise<UpdateJobByIdMutation> {
      return withWrapper((wrappedRequestHeaders) => client.request<UpdateJobByIdMutation>(UpdateJobByIdDocument, variables, {...requestHeaders, ...wrappedRequestHeaders}), 'updateJobById', 'mutation');
    },
    deleteJobByName(variables: DeleteJobByNameMutationVariables, requestHeaders?: Dom.RequestInit["headers"]): Promise<DeleteJobByNameMutation> {
      return withWrapper((wrappedRequestHeaders) => client.request<DeleteJobByNameMutation>(DeleteJobByNameDocument, variables, {...requestHeaders, ...wrappedRequestHeaders}), 'deleteJobByName', 'mutation');
    },
    deleteJobByID(variables: DeleteJobByIdMutationVariables, requestHeaders?: Dom.RequestInit["headers"]): Promise<DeleteJobByIdMutation> {
      return withWrapper((wrappedRequestHeaders) => client.request<DeleteJobByIdMutation>(DeleteJobByIdDocument, variables, {...requestHeaders, ...wrappedRequestHeaders}), 'deleteJobByID', 'mutation');
    },
    fetchForProcessing(variables?: FetchForProcessingMutationVariables, requestHeaders?: Dom.RequestInit["headers"]): Promise<FetchForProcessingMutation> {
      return withWrapper((wrappedRequestHeaders) => client.request<FetchForProcessingMutation>(FetchForProcessingDocument, variables, {...requestHeaders, ...wrappedRequestHeaders}), 'fetchForProcessing', 'mutation');
    },
    commitJobs(variables: CommitJobsMutationVariables, requestHeaders?: Dom.RequestInit["headers"]): Promise<CommitJobsMutation> {
      return withWrapper((wrappedRequestHeaders) => client.request<CommitJobsMutation>(CommitJobsDocument, variables, {...requestHeaders, ...wrappedRequestHeaders}), 'commitJobs', 'mutation');
    },
    failJobs(variables: FailJobsMutationVariables, requestHeaders?: Dom.RequestInit["headers"]): Promise<FailJobsMutation> {
      return withWrapper((wrappedRequestHeaders) => client.request<FailJobsMutation>(FailJobsDocument, variables, {...requestHeaders, ...wrappedRequestHeaders}), 'failJobs', 'mutation');
    },
    retryJobs(variables: RetryJobsMutationVariables, requestHeaders?: Dom.RequestInit["headers"]): Promise<RetryJobsMutation> {
      return withWrapper((wrappedRequestHeaders) => client.request<RetryJobsMutation>(RetryJobsDocument, variables, {...requestHeaders, ...wrappedRequestHeaders}), 'retryJobs', 'mutation');
    }
  };
}
export type Sdk = ReturnType<typeof getSdk>;