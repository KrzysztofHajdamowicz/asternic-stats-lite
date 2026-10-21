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


// This query shows the hangup cause, how many calls an
// agent hanged up, and a caller hanged up.
$query = "SELECT count(ev.event) AS num, ev.event AS action ";
$query.= "FROM queue_stats AS qs, qname AS q, qevent AS ev WHERE ";
$query.= "qs.qname = q.qname_id and qs.qevent = ev.event_id and qs.datetime >= '%s' and ";
$query.= "qs.datetime <= '%s' and q.queue IN ($queue_sql) AND ";
$query.= "ev.event IN ('COMPLETECALLER', 'COMPLETEAGENT') ";
$query.= "GROUP BY ev.event ORDER BY ev.event";

$hangup_cause["COMPLETECALLER"]=0;
$hangup_cause["COMPLETEAGENT"]=0;
$total_hangup=0;
$res = $midb->consulta($query,array($start,$end));
while($row=$midb->fetch_row($res)) {
  $hangup_cause["$row[1]"]=(int)$row[0];
  $total_hangup+=(int)$row[0];
}

$query = "SELECT qs.datetime AS datetime, q.queue AS qname, ag.agent AS qagent, "; 
$query.= "ac.event AS qevent, qs.info1 AS info1, qs.info2 AS info2,  qs.info3 AS info3 ";
$query.= "FROM queue_stats AS qs, qname AS q, qagent AS ag, qevent AS ac WHERE ";
$query.= "qs.qname = q.qname_id AND qs.qagent = ag.agent_id AND qs.qevent = ac.event_id AND ";
$query.= "qs.datetime >= '%s' AND qs.datetime <= '%s' AND ";
$query.= "q.queue IN ($queue_sql) AND ag.agent in ($agent_sql) AND ac.event IN ('COMPLETECALLER', 'COMPLETEAGENT','TRANSFER','CONNECT') ORDER BY qs.datetime";

$answer["15"]=0;
$answer["30"]=0;
$answer["45"]=0;
$answer["60"]=0;
$answer["75"]=0;
$answer["90"]=0;
$answer["91+"]=0;

$abandoned         = 0;
$transferidas      = 0;
$totaltransfers    = 0;
$total_calls       = 0;
$total_calls2      = Array();
$total_duration    = 0;
$total_hold        = 0;
$transfers         = Array();
$timeout           = 0;
$abandon_end_pos   = 0;
$abandon_start_pos = 0;
$total_hold_abandon= 0;
$timeout_end_pos   = 0;
$timeout_start_pos = 0;
$total_hold_timeout= 0;
$record            = Array();
$total_hold2       = Array();
$total_time2       = Array();
$grandtotal_hold   = 0;
$grandtotal_time   = 0;
$grandtotal_calls  = 0;
$total_calls_queue = Array();

$res = $midb->consulta($query,array($start,$end));
if($res) {
    while($row=$midb->fetch_row($res)) {
        if($row[3] <> "TRANSFER" && $row[3]<>"CONNECT") {
            $total_hold     += (int)$row[4];
            $total_duration += (int)$row[5];
            $total_calls++;
            $total_calls_queue["$row[1]"] = ($total_calls_queue["$row[1]"] ?? 0) + 1;
        } elseif($row[3]=="TRANSFER") {
            $transferidas++;
        }
        if($row[3]=="CONNECT") {

            if ($row[4] >=0 && $row[4] <= 15) {
                $answer["15"]++;
            }

            if ($row[4] >=16 && $row[4] <= 30) {
                $answer["30"]++;
            }

            if ($row[4] >=31 && $row[4] <= 45) {
              $answer["45"]++;
            }

            if ($row[4] >=46 && $row[4] <= 60) {
              $answer["60"]++;
            }

            if ($row[4] >=61 && $row[4] <= 75) {
              $answer["75"]++;
            }

            if ($row[4] >=76 && $row[4] <= 90) {
              $answer["90"]++;
            }

            if ($row[4] >=91) {
              $answer["91+"]++;
            }
        }
    }
} 

if($total_calls > 0) {
    ksort($answer);
    $average_hold     = $total_hold     / $total_calls;
    $average_duration = $total_duration / $total_calls;
    $average_hold     = number_format($average_hold     , 2);
    $average_duration = number_format($average_duration , 2);
} else {
    // There were no calls
    $average_hold = 0;
    $average_duration = 0;
}

$total_duration_print = seconds2minutes($total_duration);
// TRANSFERS
$query = "SELECT ag.agent AS agent, qs.info1 AS info1,  qs.info2 AS info2 ";
$query.= "FROM  queue_stats AS qs, qevent AS ac, qagent as ag, qname As q WHERE qs.qevent = ac.event_id ";
$query.= "AND qs.qname = q.qname_id AND ag.agent_id = qs.qagent AND qs.datetime >= '%s' ";
$query.= "AND qs.datetime <= '%s' AND  q.queue IN ($queue_sql)  AND ag.agent in ($agent_sql) AND  ac.event = 'TRANSFER'";


$res = $midb->consulta($query,array($start,$end));
if($res) {
    while($row=$midb->fetch_row($res)) {
        $keytra = "$row[0]^$row[1]@$row[2]";
        $transfers["$keytra"] = ($transfers["$keytra"] ?? 0) + 1;
        $totaltransfers++;
    }
} else {
   $totaltransfers=0;
}

// ABANDONED CALLS
$query = "SELECT  ac.event AS action,  qs.info1 AS info1,  qs.info2 AS info2,  qs.info3 AS info3 ";
$query.= "FROM  queue_stats AS qs, qevent AS ac, qname As q, qagent as ag WHERE ";
$query.= "qs.qevent = ac.event_id AND qs.qname = q.qname_id AND qs.datetime >= '%s' AND ";
$query.= "qs.datetime <= '%s' AND  q.queue IN ($queue_sql)  AND ag.agent in ($agent_sql) AND  ac.event IN ('ABANDON', 'EXITWITHTIMEOUT', 'TRANSFER') ";
$query.= "ORDER BY  ac.event,  qs.info3";

$res = $midb->consulta($query,array($start,$end));

while($row=$midb->fetch_row($res)) {

    if($row[0]=="ABANDON") {
        $abandoned++;
        $abandon_end_pos+=(int)$row[1];
        $abandon_start_pos+=(int)$row[2];
        $total_hold_abandon+=(int)$row[3];
    }
    if($row[0]=="EXITWITHTIMEOUT") {
        $timeout++;
        $timeout_end_pos+=(int)$row[1];
        $timeout_start_pos+=(int)$row[2];
        $total_hold_timeout+=(int)$row[3];
    }
}

if($abandoned > 0) {
    $abandon_average_hold = $total_hold_abandon / $abandoned;
    $abandon_average_hold = number_format($abandon_average_hold,2);

    $abandon_average_start = floor($abandon_start_pos / $abandoned);
    $abandon_average_start = number_format($abandon_average_start,2);

    $abandon_average_end = floor($abandon_end_pos / $abandoned);
    $abandon_average_end = number_format($abandon_average_end,2);
} else {
    $abandoned = 0;
    $abandon_average_hold  = 0;
    $abandon_average_start = 0;
    $abandon_average_end   = 0;
}

// This query shows every call for agents, we collect into a named array the values of holdtime and calltime
$query = "SELECT qs.datetime AS datetime, q.queue AS qname, ag.agent AS qagent, ac.event AS qevent, ";
$query.= "qs.info1 AS info1, qs.info2 AS info2, qs.info3 AS info3  FROM queue_stats AS qs, qname AS q, ";
$query.= "qagent AS ag, qevent AS ac WHERE qs.qname = q.qname_id AND qs.qagent = ag.agent_id AND ";
$query.= "qs.qevent = ac.event_id AND qs.datetime >= '%s' AND qs.datetime <= '%s' AND ";
$query.= "q.queue IN ($queue_sql) AND ag.agent in ($agent_sql) AND ac.event IN ('COMPLETECALLER', 'COMPLETEAGENT') ORDER BY ag.agent";

$res = $midb->consulta($query,array($start,$end));
while($row=$midb->fetch_row($res)) {
    $total_calls2["$row[2]"] = ($total_calls2["$row[2]"] ?? 0) + 1;
    $record["$row[2]"][]=$row[0]."|".$row[1]."|".$row[3]."|".$row[4];
    $total_hold2["$row[2]"] = ($total_hold2["$row[2]"] ?? 0) + (int)$row[4];
    $total_time2["$row[2]"] = ($total_time2["$row[2]"] ?? 0) + (int)$row[5];
    $grandtotal_hold+=(int)$row[4];
    $grandtotal_time+=(int)$row[5];
    $grandtotal_calls++;
}

$start_parts = preg_split("/ /", $start);
$end_parts   = preg_split("/ /", $end);

$cover_pdf = $lang["$language"]['queue'].": ".$queue."\n";
$cover_pdf.= $lang["$language"]['start'].": ".$start_parts[0]."\n";
$cover_pdf.= $lang["$language"]['end'].": ".$end_parts[0]."\n";
$cover_pdf.= $lang["$language"]['period'].": ".$period." ".$lang["$language"]['days']."\n\n";
$cover_pdf.= $lang["$language"]['answered_calls'].": ".$total_calls." ".$lang["$language"]['calls']."\n";
$cover_pdf.= $lang["$language"]['avg_calltime'].": ".$average_duration." ".$lang["$language"]['secs']."\n";
$cover_pdf.= $lang["$language"]['total'].": ".$total_duration_print." ".$lang["$language"]['minutes']."\n";
$cover_pdf.= $lang["$language"]['avg_holdtime'].": ".$average_hold." ".$lang["$language"]['secs']."\n";

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
        <h2><?php echo $lang["$language"]['answered_calls']?></h2>
        <dl class="kv">
            <dt><?php echo $lang["$language"]['answered_calls']?></dt>
            <dd><?php echo $total_calls?> <?php echo $lang["$language"]['calls']?></dd>
            <dt><?php echo $lang["$language"]['transferred_calls']?></dt>
            <dd><?php echo $transferidas?> <?php echo $lang["$language"]['calls']?></dd>
            <dt><?php echo $lang["$language"]['avg_calltime']?></dt>
            <dd><?php echo $average_duration?> <?php echo $lang["$language"]['secs']?></dd>
            <dt><?php echo $lang["$language"]['total']?> <?php echo $lang["$language"]['calltime']?></dt>
            <dd><?php echo $total_duration_print?> <?php echo $lang["$language"]['minutes']?></dd>
            <dt><?php echo $lang["$language"]['avg_holdtime']?></dt>
            <dd><?php echo $average_hold?> <?php echo $lang["$language"]['secs']?></dd>
        </dl>
    </section>
</div>

<section class="report" id="1">
    <h2><?php echo $lang["$language"]['answered_calls_by_agent']?></h2>
    <div class="table-wrap">
    <table class="data sortable" id="table1">
        <thead>
        <tr>
            <th><?php echo $lang["$language"]['agent']?></th>
            <th><?php echo $lang["$language"]['Calls']?></th>
            <th><?php echo $lang["$language"]['percent']?> <?php echo $lang["$language"]['Calls']?></th>
            <th><?php echo $lang["$language"]['calltime']?></th>
            <th><?php echo $lang["$language"]['percent']?> <?php echo $lang["$language"]['calltime']?></th>
            <th><?php echo $lang["$language"]['avg']?> <?php echo $lang["$language"]['calltime']?></th>
            <th><?php echo $lang["$language"]['holdtime']?></th>
            <th><?php echo $lang["$language"]['avg']?> <?php echo $lang["$language"]['holdtime']?></th>
        </tr>
        </thead>
        <tbody>
        <?php
        $header_pdf=array($lang["$language"]['agent'],$lang["$language"]['Calls'],$lang["$language"]['percent'],$lang["$language"]['calltime'],$lang["$language"]['percent'],$lang["$language"]['avg'],$lang["$language"]['holdtime'],$lang["$language"]['avg']);
        $width_pdf=array(25,23,23,23,23,25,25,20);
        $title_pdf=$lang["$language"]['answered_calls_by_agent'];

        $chart_agents = array();
        $chart_time   = array();
        $chart_calls  = array();
        $data_pdf = array();
        foreach($total_calls2 as $agent=>$val) {
            $chart_agents[] = $agent;
            $chart_time[]   = $total_time2["$agent"];
            $chart_calls[]  = $val;

            $time_print = seconds2minutes($total_time2["$agent"]);
            $avg_time = $total_time2["$agent"] / $val;
            $avg_time = round($avg_time,2);
            $avg_print = seconds2minutes($avg_time);

            if($grandtotal_calls > 0) {
                $percentage_calls = $val * 100 / $grandtotal_calls;
            } else {
                $percentage_calls = 0;
            }
            $percentage_calls = number_format($percentage_calls,2);
            if($grandtotal_time > 0) {
                $percentage_time = $total_time2["$agent"] * 100 / $grandtotal_time;
            } else {
                $percentage_time = 0;
            }
            $percentage_time = number_format($percentage_time,2);
            $avg_hold = $total_hold2["$agent"] / $val;
            $avg_hold = number_format($avg_hold,2);

            echo "<tr>\n";
            echo "<td>".h($agent)."</td>\n";
            echo "<td>$val</td>\n";
            echo "<td>$percentage_calls ".$lang["$language"]['percent']."</td>\n";
            echo "<td>$time_print ".$lang["$language"]['minutes']."</td>\n";
            echo "<td>$percentage_time ".$lang["$language"]['percent']."</td>\n";
            echo "<td>$avg_print ".$lang["$language"]['minutes']."</td>\n";
            echo "<td>".$total_hold2["$agent"]." ".$lang["$language"]['secs']."</td>\n";
            echo "<td>$avg_hold ".$lang["$language"]['secs']."</td>\n";
            echo "</tr>\n";

            $linea_pdf = array($agent,$val,"$percentage_calls ".$lang["$language"]['percent'],$total_time2["$agent"],"$percentage_time ".$lang["$language"]['percent'],"$avg_time ".$lang["$language"]['secs'],$total_hold2["$agent"]." ".$lang["$language"]['secs'], "$avg_hold ".$lang["$language"]['secs']);
            $data_pdf[]=$linea_pdf;
        }
        ?>
        </tbody>
    </table>
    </div>
    <?php
    if(count($total_calls2)>0) {
        print_exports($header_pdf,$data_pdf,$width_pdf,$title_pdf,$cover_pdf);
        echo "<div class='charts'>\n";
        chart_bar("chart1", $lang["$language"]['total_time_agent'], $chart_agents, array($lang["$language"]['seconds']=>$chart_time));
        chart_bar("chart2", $lang["$language"]['no_calls_agent'], $chart_agents, array($lang["$language"]['Calls']=>$chart_calls));
        echo "</div>\n";
    }
    ?>
</section>

<section class="report" id="2">
    <h2><?php echo $lang["$language"]['call_response']?></h2>
    <div class="split">
    <div class="table-wrap">
    <table class="data sortable" id="table2">
        <thead>
        <tr>
            <th><?php echo $lang["$language"]['answer']?></th>
            <th><?php echo $lang["$language"]['count']?></th>
            <th><?php echo $lang["$language"]['delta']?></th>
            <th><?php echo $lang["$language"]['percent']?></th>
        </tr>
        </thead>
        <tbody>
        <?php
        $countrow=0;
        $partial_total = 0;
        $chart_labels = array();
        $chart_values = array();
        $total_y_transfer = $answer['15'] + $answer['30'] +  $answer['45'] + $answer['60'] +  $answer['75'] + $answer['90'] +  $answer['91+'];
        foreach($answer as $key=>$val) {
            $chart_labels[] = "$key ".$lang["$language"]['secs'];
            $chart_values[] = $val;
            $delta = $val;
            if($delta > 0) { $delta = "+".$delta;}
            $partial_total += $val;
            if($total_y_transfer > 0) {
                $percent=$partial_total*100/$total_y_transfer;
            } else {
                $percent = 0;
            }
            $percent=number_format($percent,2);
            if($countrow==0) { $delta = ""; }
            echo "<tr>\n";
            echo "<td>".$lang["$language"]['within']."$key ".$lang["$language"]['secs']."</td>\n";
            echo "<td>$partial_total ".$lang["$language"]['calls']."</td>\n";
            echo "<td>$delta</td>\n";
            echo "<td>$percent ".$lang["$language"]['percent']."</td>\n";
            echo "</tr>\n";
            $countrow++;
        }
        ?>
        </tbody>
    </table>
    </div>
    <?php
    chart_bar("chart3", $lang["$language"]['call_response'], $chart_labels, array($lang["$language"]['Calls']=>$chart_values));
    ?>
    </div>
</section>

<section class="report" id="3">
    <h2><?php echo $lang["$language"]['answered_calls_by_queue']?></h2>
    <div class="split">
    <div class="table-wrap">
    <table class="data sortable" id="table3">
        <thead>
        <tr>
            <th><?php echo $lang["$language"]['queue']?></th>
            <th><?php echo $lang["$language"]['count']?></th>
            <th><?php echo $lang["$language"]['percent']?></th>
        </tr>
        </thead>
        <tbody>
        <?php
        $countrow=0;
        $chart_labels = array();
        $chart_values = array();
        if(count($total_calls_queue)==0) {
            $total_calls_queue[""]=0;
        }
        asort($total_calls_queue);
        foreach($total_calls_queue as $key=>$val) {
            if($total_calls>0) {
                $percent = $val * 100 / $total_calls;
            } else {
                $percent=0;
            }
            $percent =number_format($percent,2);
            echo "<tr><td>".h($key)."</td><td>$val ".$lang["$language"]['calls']."</td><td>$percent %</td></tr>\n";
            $countrow++;
            $chart_labels[] = $key;
            $chart_values[] = $val;
        }
        ?>
        </tbody>
    </table>
    </div>
    <?php
    if ($countrow>1) {
        chart_bar("chart4", $lang["$language"]['answered_calls_by_queue'], $chart_labels, array($lang["$language"]['Calls']=>$chart_values));
    }
    ?>
    </div>
</section>

<section class="report" id="4">
    <h2><?php echo $lang["$language"]['disconnect_cause']?></h2>
    <div class="split">
    <div class="table-wrap">
    <table class="data sortable" id="table4">
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
            'COMPLETEAGENT'  => $lang["$language"]['agent_hungup'],
            'COMPLETECALLER' => $lang["$language"]['caller_hungup'],
        );
        foreach($causes as $event=>$label) {
            if($total_hangup > 0) {
                $percent = $hangup_cause[$event]*100/$total_hangup;
            } else {
                $percent = 0;
            }
            $percent = number_format($percent,2);
            echo "<tr><td>$label</td><td>".$hangup_cause[$event]." ".$lang["$language"]['calls']."</td><td>$percent ".$lang["$language"]['percent']."</td></tr>\n";
        }
        ?>
        </tbody>
    </table>
    </div>
    <?php
    chart_bar("chart5", $lang["$language"]['disconnect_cause'],
        array($lang["$language"]['agent'], $lang["$language"]['caller']),
        array($lang["$language"]['Calls']=>array($hangup_cause["COMPLETEAGENT"], $hangup_cause["COMPLETECALLER"])));
    ?>
    </div>
</section>

<?php if($totaltransfers>0) { ?>
<section class="report">
    <h2><?php echo $lang["$language"]['transfers']?></h2>
    <div class="table-wrap">
    <table class="data sortable" id="table5">
        <thead>
        <tr>
            <th><?php echo $lang["$language"]['agent']?></th>
            <th><?php echo $lang["$language"]['to']?></th>
            <th><?php echo $lang["$language"]['count']?></th>
        </tr>
        </thead>
        <tbody>
        <?php
        foreach($transfers as $key=>$val) {
            $partes = preg_split("/\^/",$key);
            echo "<tr><td>".h($partes[0])."</td><td>".h($partes[1])."</td><td>$val</td></tr>\n";
        }
        ?>
        </tbody>
    </table>
    </div>
</section>
<?php } ?>
<?php include("footer.php"); ?>
