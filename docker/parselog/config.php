<?php
// Container configuration: replaces parselog/config.php in the parselog image.
require_once("dblib.php");
require_once("misc.php");

$queue_log_dir  = rtrim(getenv('QUEUE_LOG_DIR') ?: '/var/log/asterisk', '/') . '/';
$queue_log_file = getenv('QUEUE_LOG_FILE') ?: 'queue_log';

$dbhost = getenv('dbhost') ?: 'database';
$dbname = getenv('dbname') ?: 'qstatslite';
$dbuser = getenv('dbuser') ?: 'qstatslite';
$dbpass = getenv('dbpass') ?: '';

$midb = new dbcon($dbhost, $dbuser, $dbpass, $dbname, true);
$self = $_SERVER['PHP_SELF'];

$DB_DEBUG = false;

?>
