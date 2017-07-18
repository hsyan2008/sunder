<?php
/* Connect to an ODBC database using driver invocation */
$dsn = 'mysql:dbname=redmaple;host=127.0.0.1;port=13306';
$user = 'test';
$password = 'test';

//连接
try {
    $dbh = new PDO($dsn, $user, $password);
} catch (PDOException $e) {
    echo 'Connection failed: ' . $e->getMessage();
}

//普通查询
$sql = 'SELECT name  FROM tasks ORDER BY name';
foreach ($dbh->query($sql) as $row) {
    // print $row['name'] . "\n";
}

//普通插入
$sql = 'insert into tasks set name = "bbb"';
$dbh->query($sql);
// print_r($dbh->lastInsertId());

//pdo查询
$sql = 'SELECT * FROM tasks WHERE id = :id';
$sth = $dbh->prepare($sql, array(PDO::ATTR_CURSOR => PDO::CURSOR_FWDONLY));
$sth->execute(array(':id' => 1));
$red = $sth->fetchAll();
// print_r($red);

//事务回滚
$dbh->beginTransaction();
$sql = 'SELECT * FROM tasks WHERE id = :id';
$sth = $dbh->prepare($sql, array(PDO::ATTR_CURSOR => PDO::CURSOR_FWDONLY));
$sth->execute(array(':id' => 1));
$red = $sth->fetchAll();
// print_r($red);
$dbh->rollBack();

//事务提交
$dbh->beginTransaction();
$sql = 'SELECT * FROM tasks WHERE id = :id';
$sth = $dbh->prepare($sql, array(PDO::ATTR_CURSOR => PDO::CURSOR_FWDONLY));
$sth->execute(array(':id' => 1));
$red = $sth->fetchAll();
// print_r($red);
$dbh->commit();


$stmt = $dbh->prepare("INSERT INTO REGISTRY (name, value) VALUES (?, ?)");
$stmt->bindParam(1, $name);
$stmt->bindParam(2, $value);
// 插入一行
$name = 'one';
$value = 1;
$stmt->execute();
