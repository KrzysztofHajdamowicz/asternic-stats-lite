<?php
/*
   Copyright 2007, 2020 Nicolás Gudiño

   This file is part of Asternic Call Center Stats.

    Asternic Call Center Stats is free software: you can redistribute it 
    and/or modify it under the terms of the GNU General Public License as 
    published by the Free Software Foundation, either version 3 of the 
    License, or (at your option) any later version.

    Asternic Call Center Stats is distributed in the hope that it will be 
    useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with Asternic Call Center Stats.  If not, see 
    <http://www.gnu.org/licenses/>.
*/
?>
<header class="site-header" id="0">
<div class="brand">Asternic <span>Call Center Stats</span></div>
<nav class="tabs" aria-label="Reports">
<?php
$menu = array(
    "index.php"        => $lang["$language"]['menu_home'],
    "answered.php"     => $lang["$language"]['menu_answered'],
    "unanswered.php"   => $lang["$language"]['menu_unanswered'],
    "distribution.php" => $lang["$language"]['menu_distribution'],
    "realtime.php"     => "Realtime",
);

// Section links of the current report
$anchor = Array();
if(basename($self)=="answered.php") {
    $anchor[]=$lang["$language"]['answered_calls_by_agent'];
    $anchor[]=$lang["$language"]['call_response'];
    $anchor[]=$lang["$language"]['answered_calls_by_queue'];
    $anchor[]=$lang["$language"]['disconnect_cause'];
} elseif (basename($self) =="unanswered.php") {
    $anchor[]=$lang["$language"]['disconnect_cause'];
    $anchor[]=$lang["$language"]['unanswered_calls_qu'];
} elseif (basename($self) =="distribution.php") {
    $anchor[]=$lang["$language"]['call_distrib_day'];
    $anchor[]=$lang["$language"]['call_distrib_hour'];
    $anchor[]=$lang["$language"]['call_distrib_week'];
}

foreach($menu as $link=>$label) {
    if(basename($self)==$link) {
        echo "<a href='$link' aria-current='page'>".h($label)."</a>\n";
    } elseif(isset($_SESSION['QSTATS']['start'])) {
        // Reports need a selection made on the home page first
        echo "<a href='$link'>".h($label)."</a>\n";
    }
}
?>
</nav>
<?php if(count($anchor)>0) { ?>
<nav class="sections" aria-label="Sections">
<?php
foreach ($anchor as $contador=>$item) {
    echo "<a href='#".($contador+1)."'>".h($item)."</a>\n";
}
?>
</nav>
<?php } ?>
</header>
