<?php
// Container configuration: replaces html/config.php in the web image.
// Everything comes from the environment (see the env vars in README.md).
require_once("dblib.php");
require_once("misc.php");

// Credentials for MYSQL database
$dbhost = getenv('dbhost') ?: 'database';
$dbname = getenv('dbname') ?: 'qstatslite';
$dbuser = getenv('dbuser') ?: 'qstatslite';
$dbpass = getenv('dbpass') ?: '';

// Credentials for AMI (for the realtime tab to work)
// See /etc/asterisk/manager.conf
$manager_host   = getenv('manager_host') ?: '127.0.0.1';
$manager_user   = getenv('manager_user') ?: '';
$manager_secret = getenv('manager_pass') ?: '';

// Available languages "es", "en", "ru", "de", "fr", "pt_BR"
$language = getenv('language') ?: 'en';
if (!in_array($language, array('es', 'en', 'ru', 'de', 'fr', 'pt_BR'), true)) {
    $language = 'en';
}

require_once("lang/$language.php");

$midb = new dbcon($dbhost, $dbuser, $dbpass, $dbname, true);

$self = $_SERVER['PHP_SELF'];

$DB_DEBUG = false;

session_start();
header('content-type: text/html; charset: utf-8');

?>
