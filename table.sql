use megnet;
DROP TABLE IF EXISTS  t_torrent_info;/*SkipError*/
CREATE TABLE t_torrent_info(
    id BIGINT NOT NULL AUTO_INCREMENT  COMMENT '' ,
    hash VARCHAR(64) NOT NULL   COMMENT '' ,
    name VARCHAR(256)    COMMENT '' ,
    discover_time datetime  DEFAULT NULL,
    discover_from VARCHAR(32)    COMMENT '' ,
    PRIMARY KEY(id),
    UNIQUE KEY (hash),
    KEY `ind_name` (`name`)
)DEFAULT CHARSET=utf8;
DROP TABLE IF EXISTS  files_table;/*SkipError*/

CREATE TABLE t_files_info(
    id BIGINT NOT NULL AUTO_INCREMENT  COMMENT '' ,
    info_id BIGINT NOT NULL COMMENT '' ,
    hash VARCHAR(64) NOT NULL   COMMENT '' ,
    file_name VARCHAR(256)    COMMENT '' ,
    size datetime  DEFAULT NULL,
    PRIMARY KEY(id),
    KEY `ind_hash` (`hash`),
    KEY `ind_file_name` (`file_name`)
)DEFAULT CHARSET=utf8;