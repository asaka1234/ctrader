exchange服务
========================
1. 相当于交易server
2. 对外提供三种接口
   1. fix api
   2. grpc
   3. rest api


rest api
----------------------
1. 提供了一些query查询接口
2. 提供了websocket接口，允许用户来订阅，并推送最新的orderbook过去
3. 注意:不是纯api, 还有用模板渲染的方式提供的web页面(资源放在web目录下, 渲染模板用的 https://github.com/gernest/hot 库)