# Lab2

## Design thoughts
- Server need to distinguish duplicate request.
  - Each Client maintains a op index, which increase when new operation.
    - Server can use this op index to distinguish same op from same client.
  - Each client should have their own id.
    - Generated randomly because different clients are hard to sync.
  - Client won't send previous request
    - It's ok for server to record latest client's id operation .
- 