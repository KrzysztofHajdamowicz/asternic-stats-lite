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


require_once("config.php");
require_once("sesvars.php");

// Date range shortcuts
$today = date('Y-m-d');
$shortcuts = array(
    'today'             => array($today, $today),
    'this_week'         => array(date('Y-m-d', strtotime('monday this week')), date('Y-m-d', strtotime('sunday this week'))),
    'this_month'        => array(date('Y-m-01'), date('Y-m-t')),
    'last_three_months' => array(date('Y-m-01', strtotime('first day of -2 months')), date('Y-m-t')),
);

$colas   = Array();
$agentes = Array();

$query = "SELECT queue FROM qname ORDER BY queue";
$res = $midb->consulta($query);
while ($row = $midb->fetch_row($res)) {
  $colas[] = $row[0];
}

$query = "SELECT agent FROM qagent ORDER BY agent";
$res = $midb->consulta($query);
while ($row = $midb->fetch_row($res)) {
  $agentes[] = $row[0];
}

// Queues and agents chosen last time, stored as 'a','b' in the session
function remove_quotes($argument)
{
    return substr($argument,1,-1);
}
$items_cola   = array_map("remove_quotes", explode(",",$queue));
$items_agente = array_map("remove_quotes", explode(",",$agent));

// No agent chosen yet: preselect all of them
if($agent == "''") {
    $items_agente = $agentes;
}

// Prints the "available" and "selected" lists of a dual list box
function picker($name, $label, $all, $chosen) {
    global $lang, $language;
    $available = array();
    $selected  = array();
    foreach($all as $item) {
        if($item == "NONE") continue;
        if(in_array($item, $chosen)) {
            $selected[] = $item;
        } else {
            $available[] = $item;
        }
    }
    echo "<fieldset class='picker' data-picker>\n";
    echo "<legend>".h($label)."</legend>\n";
    echo "<label>".h($lang[$language]['available'])."\n";
    echo "<select multiple size='8' data-from>\n";
    foreach($available as $item) {
        echo "<option value=\"".h("'$item'")."\">".h($item)."</option>\n";
    }
    echo "</select></label>\n";
    echo "<div class='picker-buttons'>\n";
    echo "<button type='button' data-move='add' title='Add selected' aria-label='Add selected'>&rsaquo;</button>\n";
    echo "<button type='button' data-move='remove' title='Remove selected' aria-label='Remove selected'>&lsaquo;</button>\n";
    echo "<button type='button' data-move='add-all' title='Add all' aria-label='Add all'>&raquo;</button>\n";
    echo "<button type='button' data-move='remove-all' title='Remove all' aria-label='Remove all'>&laquo;</button>\n";
    echo "</div>\n";
    echo "<label>".h($lang[$language]['selected'])."\n";
    echo "<select multiple size='8' name='".h($name)."[]' data-to>\n";
    foreach($selected as $item) {
        echo "<option value=\"".h("'$item'")."\">".h($item)."</option>\n";
    }
    echo "</select></label>\n";
    echo "</fieldset>\n";
}

$page_scripts = array("js/index.js");
include("header.php");
?>
<form method="post" action="answered.php" id="report-form"
      data-msg-queue="Please select at least one queue"
      data-msg-agent="Please select at least one agent"
      data-msg-date="<?php echo h($lang[$language]['invaliddate']); ?>">
<input type="hidden" name="start">
<input type="hidden" name="end">

<div class="pickers">
<?php
picker('List_Queue', $lang[$language]['select_queue'], $colas,   $items_cola);
picker('List_Agent', $lang[$language]['select_agent'], $agentes, $items_agente);
?>
</div>

<fieldset class="timeframe">
<legend><?php echo h($lang[$language]['select_timeframe']); ?></legend>
<p class="shortcuts"><?php echo h($lang[$language]['shortcuts']); ?>:
<?php foreach($shortcuts as $key=>$range) { ?>
    <button type="button" class="link" data-start="<?php echo $range[0]; ?>" data-end="<?php echo $range[1]; ?>"><?php echo h($lang[$language][$key]); ?></button>
<?php } ?>
</p>
<p class="dates">
<label><?php echo h($lang[$language]['start']); ?>
    <input type="date" id="date-start" value="<?php echo h(substr($start,0,10)); ?>" required></label>
<label><?php echo h($lang[$language]['end']); ?>
    <input type="date" id="date-end" value="<?php echo h(substr($end,0,10)); ?>" required></label>
</p>
</fieldset>

<p><button type="submit" class="primary"><?php echo h($lang[$language]['display_report']); ?></button></p>
</form>
<?php include("footer.php"); ?>
