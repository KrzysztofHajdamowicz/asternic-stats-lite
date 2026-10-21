<?php
/*
   Common page header. Include after config.php (and sesvars.php).

   $page_scripts: optional list of extra JavaScript files for this page.
*/
?>
<!doctype html>
<html lang="<?php echo h(substr($language,0,2)); ?>">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Asternic Call Center Stats</title>
  <meta name="description" content="Asternic Call Center Stats Lite Reports">
  <link rel="stylesheet" href="css/basic.css">
  <link rel="stylesheet" href="css/tab.css">
  <link rel="stylesheet" href="css/table.css">
  <link rel="stylesheet" href="css/fixed-all.css">
<?php foreach (($page_scripts ?? array()) as $script) { ?>
  <script src="<?php echo h($script); ?>" defer></script>
<?php } ?>
</head>
<body>
<?php include("menu.php"); ?>
<div id="main">
<div id="contents">
