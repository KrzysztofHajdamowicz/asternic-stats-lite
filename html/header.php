<?php
/*
   Copyright 2026 Krzysztof Hajdamowicz
   Part of Asternic Call Center Stats Lite, GPL-3.0-or-later (see LICENSE).

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
  <meta name="color-scheme" content="light dark">
  <link rel="stylesheet" href="css/app.css">
<?php foreach (($page_scripts ?? array()) as $script) { ?>
  <script src="<?php echo h($script); ?>" defer></script>
<?php } ?>
</head>
<body>
<?php include("menu.php"); ?>
<main class="page">
