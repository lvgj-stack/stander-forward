create table chains
(
    id         bigint unsigned auto_increment
        primary key,
    created_at datetime(3)  null,
    updated_at datetime(3)  null,
    deleted_at datetime(3)  null,
    chain_name varchar(255) null,
    ip         varchar(255) null,
    port       int          null,
    protocol   varchar(255) null,
    `key`      varchar(255) null,
    node_id    bigint       not null
);

create index idx_chains_deleted_at
    on chains (deleted_at);

create table nodes
(
    id         bigint unsigned auto_increment
        primary key,
    created_at datetime(3)                 null,
    updated_at datetime(3)                 null,
    deleted_at datetime(3)                 null,
    node_name  varchar(255)                null,
    ip         varchar(255)                null,
    port       int                         null,
    `key`      varchar(255)                null,
    status     varchar(255) default '8123' null,
    node_type  varchar(255)                null,
    ipv4       varchar(255)                null,
    ipv6       varchar(255)                null
);

create table permission
(
    id          int          null,
    name        varchar(255) null,
    code        varchar(50)  null,
    type        varchar(255) null,
    parentId    int          null,
    path        varchar(255) null,
    redirect    varchar(255) null,
    icon        varchar(255) null,
    component   varchar(255) null,
    layout      varchar(255) null,
    keepAlive   tinyint      null,
    method      varchar(255) null,
    description varchar(255) null,
    `show`      tinyint      null,
    enable      tinyint      null,
    `order`     int          null
);

create table profile
(
    id       int          null,
    gender   int          null,
    avatar   varchar(255) null,
    address  varchar(255) null,
    email    varchar(255) null,
    userId   int          null,
    nickName varchar(10)  null
);

create table role
(
    id     int         null,
    code   varchar(50) null,
    name   varchar(50) null,
    enable tinyint     null
);

create table role_permissions_permission
(
    roleId       int null,
    permissionId int null
);

create table rules
(
    id          bigint unsigned auto_increment
        primary key,
    created_at  datetime(3)      null,
    updated_at  datetime(3)      null,
    deleted_at  datetime(3)      null,
    rule_name   varchar(255)     null,
    node_id     bigint           null,
    chain_id    bigint           null,
    listen_port int              null,
    remote_addr varchar(255)     null,
    protocol    varchar(255)     null,
    traffic     bigint default 0 not null
);

create table user
(
    id         int          null,
    username   varchar(50)  null,
    password   varchar(255) null,
    enable     tinyint      null,
    createTime datetime(6)  null,
    updateTime datetime(6)  null
);

create table user_roles_role
(
    userId int null,
    roleId int null
);


INSERT INTO `permission` VALUES (1,'资源管理','Resource_Mgt','MENU',2,'/pms/resource',NULL,'i-fe:list','/src/views/pms/resource/index.vue',NULL,NULL,NULL,NULL,1,1,1),(2,'系统管理','SysMgt','MENU',NULL,NULL,NULL,'i-fe:grid',NULL,NULL,NULL,NULL,NULL,1,1,2),(3,'角色管理','RoleMgt','MENU',2,'/pms/role',NULL,'i-fe:user-check','/src/views/pms/role/index.vue',NULL,NULL,NULL,NULL,1,1,2),(4,'用户管理','UserMgt','MENU',2,'/pms/user',NULL,'i-fe:user','/src/views/pms/user/index.vue',NULL,1,NULL,NULL,1,1,3),(5,'分配用户','RoleUser','MENU',3,'/pms/role/user/:roleId',NULL,'i-fe:user-plus','/src/views/pms/role/role-user.vue','full',NULL,NULL,NULL,0,1,1),(6,'业务示例','Demo','MENU',NULL,NULL,NULL,'i-fe:grid',NULL,NULL,NULL,NULL,NULL,1,1,1),(7,'图片上传','ImgUpload','MENU',6,'/demo/upload',NULL,'i-fe:image','/src/views/demo/upload/index.vue','',1,NULL,NULL,1,1,2),(8,'个人资料','UserProfile','MENU',NULL,'/profile',NULL,'i-fe:user','/src/views/profile/index.vue',NULL,NULL,NULL,NULL,0,1,99),(9,'基础功能','Base','MENU',NULL,'',NULL,'i-fe:grid',NULL,'',NULL,NULL,NULL,1,1,0),(10,'基础组件','BaseComponents','MENU',9,'/base/components',NULL,'i-me:awesome','/src/views/base/index.vue',NULL,NULL,NULL,NULL,1,1,1),(11,'Unocss','Unocss','MENU',9,'/base/unocss',NULL,'i-me:awesome','/src/views/base/unocss.vue',NULL,NULL,NULL,NULL,1,1,2),(12,'KeepAlive','KeepAlive','MENU',9,'/base/keep-alive',NULL,'i-me:awesome','/src/views/base/keep-alive.vue',NULL,1,NULL,NULL,1,1,3),(13,'创建新用户','AddUser','BUTTON',4,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,1,1,1),(14,'图标 Icon','Icon','MENU',9,'/base/icon',NULL,'i-fe:feather','/src/views/base/unocss-icon.vue','',NULL,NULL,NULL,1,1,0),(15,'MeModal','TestModal','MENU',9,'/testModal',NULL,'i-me:dialog','/src/views/base/test-modal.vue',NULL,NULL,NULL,NULL,1,1,5),(21,'服务器','Server','MENU',NULL,'/servers','','i-fe:server','/src/views/servers/index.vue','',0,'','',1,1,10),(22,'转发链','Chain','MENU',NULL,'/chains','','i-fe:fast-forward','/src/views/chains/index.vue','',0,'','',1,1,11),(23,'转发规则','Rule','MENU',NULL,'/rules','','i-fe:list','/src/views/rule/index.vue','',0,'/src/views/rule/index.vue','',1,1,12);
INSERT INTO `profile` VALUES (1,0,'https://wpimg.wallstcn.com/f778738c-e4f8-4870-b634-56703b4acafe.gif?imageView2/1/w/80/h/80','123123',NULL,1,'admin'),(3,0,'https://api.dicebear.com/7.x/miniavs/svg?seed=412','asdas','lvgj1998@gmail.com',3,'user01');
INSERT INTO `role` VALUES (1,'SUPER_ADMIN','超级管理员',1),(2,'ROLE_QA','质检员',1),(4,'USER','普通用户',1);
INSERT INTO `role_permissions_permission` VALUES (2,1),(2,2),(2,3),(2,4),(2,5),(2,9),(2,10),(2,11),(2,12),(2,14),(2,15),(4,8),(4,21),(4,22),(4,23);
INSERT INTO `user` VALUES (1,'admin','53d3c4c5c5f07891133f49250f6f13d9',1,'2023-11-18 16:18:59.150632','2024-10-12 08:07:01.652188'),(3,'user01','53d3c4c5c5f07891133f49250f6f13d9',1,'2024-10-13 22:55:16.601588','2024-10-13 22:55:16.601588');
INSERT INTO `user_roles_role` VALUES (1,1),(1,2),(1,4),(3,4);