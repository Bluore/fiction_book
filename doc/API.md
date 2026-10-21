# API 接口文档

| 模块 | 接口 |
| --- | --- |
| 系统 | 健康检查(HealthCheck) |
| 认证 | 用户登录(Login)<br>获取邮箱验证码(EmailVerify)<br>忘记密码(ForgetPassword) |
| 用户管理 | 创建用户(CreateUser)<br>获取用户列表(ListUsers)<br>获取用户信息(GetUser)<br>获取当前用户详情信息(GetMyUserInfo)<br>更新用户信息(UpdateUser)<br>更新用户密码(UpdateUserPassword)<br>更新用户邮箱(UpdateUserEmail)<br>删除用户(DeleteUser)<br>上传头像(UploadAvatar) |
| 书籍管理 | 创建书籍(CreateBook)<br>获取书籍(GetBook)<br>获取书籍列表(BookList)<br>上传书籍封面(UploadBookCoverImage)<br>获取书籍章节列表(GetBookChapterList)<br>获取每日热度榜单(GetBookListThermalDaily)<br>获取每周热度榜单(GetBookListThermalWeekly)<br>获取每月热度榜单(GetBookListThermalMonthly)<br>获取用户创作的书籍(GetBookListByOwner)<br>获取用户最近读书的历史记录(GetUserReadBookHistoryRecently)<br>更新书籍信息(UpdateBookInformation)<br>搜索书籍(SearchBooksV2) |
| 章节管理 | 创建书籍章节(AddBookChapter)<br>获取章节内容游客版(GuestGetBookChapter)<br>获取章节内容认证版(GetBookChapter)<br>修改编辑小说章节(UpdateBookChapter)<br>删除章节(DeleteBookChapter)<br>获取章节信息(GetBookChapterInfo)<br>搜索章节(SearchBookChaptersV2) |
| 消费管理 | 购买书籍(BuyBook)<br>购买章节(BuyBookChapter)<br>获取VIP商品列表(ShopListOfVIP)<br>购买VIP(BuyVIP)<br>升级VIP(UpgradeVIP) |
| 资产管理 | 获取用户资产(GetUserAsset)<br>获取用户已购买的书籍列表(GetUserPurchasedBook)<br>判断用户是否已购买本书籍(CheckUserPurchasedBook)<br>获取用户已购买的章节列表(GetUserPurchasedChapter)<br>判断用户是否已购买本章节(CheckUserPurchasedChapter) |
| 书架管理 | 获取自己书架(ListMineBookshelf)<br>获取用户书架(ListUserBookshelf)<br>添加到书架(CreateUserBookshelf)<br>从书架中删除(DeleteUserBookshelf) |
| 历史管理 | 获取用户浏览历史(ListUserHistory) |
| 订单管理 | 获取用户订单(GetOrderByID)<br>获取用户订单列表(GetOrderList)<br>创建用户充值订单(CreateOrder) |
| 支付管理 | 支付宝异步通知(AlipayNotify) |
