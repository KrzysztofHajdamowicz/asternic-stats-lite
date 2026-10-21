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
include("sesvars.php");
$page_scripts = array("js/sortable.js", "js/vendor/chart.umd.min.js", "js/charts.js");
include("header.php");



// ABANDONED CALLS

$query = "SELECT qs.datetime AS datetime, q.queue AS qname, ag.agent AS qagent, ac.event AS qevent, ";
$query.= "qs.info1 AS info1, qs.info2 AS info2,  qs.info3 AS info3 FROM queue_stats AS qs, qname AS q, ";
$query.= "qagent AS ag, qevent AS ac WHERE qs.qname = q.qname_id AND qs.qagent = ag.agent_id AND ";
$query.= "qs.qevent = ac.event_id AND qs.datetime >= '%s' AND qs.datetime <= '%s' ";
$query.= "AND q.queue IN ($queue_sql) AND ac.event IN ('ABANDON', 'EXITWITHTIMEOUT') ORDER BY qs.datetime";
$res = $midb->consulta($query,array($start,$end));

$abandon_calls_queue = Array();
$abandoned=0;
$timeout=0;
$abandon_end_pos=0;
$abandon_start_pos=0;
$total_hold_abandon=0;

if($midb->num_rows($res)>0) {

while($row=$midb->fetch_row($res)) {

    if($row[3]=="ABANDON") {
        $abandoned++;
        $abandon_end_pos+=(int)$row[4];
        $abandon_start_pos+=(int)$row[5];
        $total_hold_abandon+=(int)$row[6];
    }
    if($row[3]=="EXITWITHTIMEOUT") {
         $timeout++;
    }
    $abandon_calls_queue["$row[1]"] = ($abandon_calls_queue["$row[1]"] ?? 0) + 1;
}

if($abandoned > 0) {
    $abandon_average_hold = $total_hold_abandon / $abandoned;
} else {
    $abandon_average_hold = 0;
}
$abandon_average_hold = number_format($abandon_average_hold,0);

if($abandoned > 0) {
    $abandon_average_start = round($abandon_start_pos / $abandoned);
} else {
    $abandon_average_start = 0;
}
$abandon_average_start = number_format($abandon_average_start,0);

if($abandoned > 0) {
    $abandon_average_end = floor($abandon_end_pos / $abandoned);
} else {
    $abandon_average_end = 0;
}
$abandon_average_end = number_format($abandon_average_end,0);

$total_abandon = $abandoned + $timeout;

} else {
     // No rows returned
    $abandoned = 0;
    $timeout = 0;
    $abandon_average_hold  = 0;
    $abandon_average_start = 0;
    $abandon_average_end   = 0;
    $total_abandon         = 0;
}


$start_parts = preg_split("/ /", $start);
$end_parts   = preg_split("/ /", $end);

?>
<div class="summary">
    <section class="card">
        <h2><?php echo $lang["$language"]['report_info']?></h2>
        <dl class="kv">
            <dt><?php echo $lang["$language"]['queue']?></dt>
            <dd><?php echo h($queue)?></dd>
            <dt><?php echo $lang["$language"]['start']?></dt>
            <dd><?php echo h($start_parts[0])?></dd>
            <dt><?php echo $lang["$language"]['end']?></dt>
            <dd><?php echo h($end_parts[0])?></dd>
            <dt><?php echo $lang["$language"]['period']?></dt>
            <dd><?php echo $period?> <?php echo $lang["$language"]['days']?></dd>
        </dl>
    </section>
    <section class="card">
        <h2><?php echo $lang["$language"]['unanswered_calls']?></h2>
        <dl class="kv">
            <dt><?php echo $lang["$language"]['number_unanswered']?></dt>
            <dd><?php echo $total_abandon?> <?php echo $lang["$language"]['calls']?></dd>
            <dt><?php echo $lang["$language"]['avg_wait_before_dis']?></dt>
            <dd><?php echo $abandon_average_hold?> <?php echo $lang["$language"]['secs']?></dd>
            <dt><?php echo $lang["$language"]['avg_queue_pos_at_dis']?></dt>
            <dd><?php echo $abandon_average_end?></dd>
            <dt><?php echo $lang["$language"]['avg_queue_start']?></dt>
            <dd><?php echo $abandon_average_start?></dd>
        </dl>
    </section>
</div>

<section class="report" id="1">
    <h2><?php echo $lang["$language"]['disconnect_cause']?></h2>
    <div class="split">
    <div class="table-wrap">
    <table class="data" id="table1">
        <thead>
        <tr>
            <th><?php echo $lang["$language"]['cause']?></th>
            <th><?php echo $lang["$language"]['count']?></th>
            <th><?php echo $lang["$language"]['percent']?></th>
        </tr>
        </thead>
        <tbody>
        <?php
        $causes = array(
            array($lang["$language"]['user_abandon'], $abandoned),
            array($lang["$language"]['timeout'], $timeout),
        );
        foreach($causes as $cause) {
            if($total_abandon > 0) {
                $percent = $cause[1]*100/$total_abandon;
            } else {
                $percent = 0;
            }
            $percent = number_format($percent,2);
            echo "<tr><td>".$cause[0]."</td><td>".$cause[1]." ".$lang["$language"]['calls']."</td><td>$percent ".$lang["$language"]['percent']."</td></tr>\n";
        }
        ?>
        </tbody>
    </table>
    </div>
    <?php
    chart_bar("chart1", $lang["$language"]['disconnect_cause'],
        array($lang["$language"]['abandon'], $lang["$language"]['timeout']),
        array($lang["$language"]['Calls']=>array($abandoned, $timeout)));
    ?>
    </div>
</section>

<?php
if(count($abandon_calls_queue)<=0) {
    $abandon_calls_queue[""]=0;
}
?>
<section class="report" id="2">
    <h2><?php echo $lang["$language"]['unanswered_calls_qu']?></h2>
    <div class="split">
    <div class="table-wrap">
    <table class="data sortable" id="table2">
        <thead>
        <tr>
            <th><?php echo $lang["$language"]['queue']?></th>
            <th><?php echo $lang["$language"]['count']?></th>
            <th><?php echo $lang["$language"]['percent']?></th>
        </tr>
        </thead>
        <tbody>
        <?php
        $chart_labels = array();
        $chart_values = array();
        asort($abandon_calls_queue);
        foreach($abandon_calls_queue as $key=>$val) {
            if($total_abandon > 0) {
                $percent = $val * 100 / $total_abandon;
            } else {
                $percent = 0;
            }
            $percent =number_format($percent,2);
            echo "<tr><td>".h($key)."</td><td>$val ".$lang["$language"]['calls']."</td><td>$percent ".$lang["$language"]['percent']."</td></tr>\n";
            $chart_labels[] = $key;
            $chart_values[] = $val;
        }
        ?>
        </tbody>
    </table>
    </div>
    <?php
    chart_bar("chart2", $lang["$language"]['unanswered_calls_qu'], $chart_labels, array($lang["$language"]['Calls']=>$chart_values));
    ?>
    </div>
</section>
<?php include("footer.php"); ?>
