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

function return_timestamp($date_string) {
    $parts = array_pad(preg_split("/-|:| /",(string)$date_string,6), 6, 0);
    list ($year,$month,$day,$hour,$min,$sec) = array_map('intval', $parts);
    $u_timestamp = mktime($hour,$min,$sec,$month,$day,$year);
    return $u_timestamp;
}

function check_queue($queue_name) {
    global $queuecache, $midb;

    if($queue_name=="") {
        return 0;
    }

    if(isset($queuecache["$queue_name"])) {
        return $queuecache["$queue_name"];
    }

    $query = "SELECT qname_id,queue FROM qname WHERE queue='%s'";
    $res = $midb->consulta($query,$queue_name);

    if($midb->num_rows($res)>0) {
        $row = $midb->fetch_row($res);
        $queuecache["$queue_name"]=$row[0];
        return $row[0];
    } else {
        $query = "INSERT INTO qname (queue) VALUES ('%s')";
        $res = $midb->consulta($query,$queue_name);
        $id = $midb->insert_id($res);
        $queuecache["$queue_name"]=$id;
        return $id;
    }
}

function check_agent($agent) {
    global $agentcache;
    global $argv, $midb, $convertlocal;

    if($agent=="") {
        return 0;
    }

    if($convertlocal) {
        if(preg_match("/^Local\/([^@]*).*/i",$agent,$matches)) {
            $agent = "Agent/".$matches[1];
        }
    }

    if(isset($agentcache["$agent"])) {
        return $agentcache["$agent"];
    }

    $query = "SELECT agent_id,agent FROM qagent WHERE agent='%s'";
    $res = $midb->consulta($query,$agent);

    if($midb->num_rows($res)>0) {
        $row = $midb->fetch_row($res);
        $agentcache["$agent"]=$row[0];
        return $row[0];
    } else {
        $query = "INSERT INTO qagent (agent) VALUES ('%s')";
        $res = $midb->consulta($query,$agent);
        $id = $midb->insert_id($res);
        $agentcache["$agent"]=$id;
        return $id;
    }
}

function procesa($linea) {

    global $event_array;
    global $last_event_ts;
    global $midb;

    if($linea === false || $linea === null) {
        return;
    }

    $linea = rtrim($linea);
    //list ($date,$uniqueid,$queue_name,$agent,$event,$data1,$data2,$data3) = preg_split("/\|/",$linea,8);
    $partes     = preg_split("/\|/",$linea,8);
    $date       = (string)array_shift($partes);
    $uniqueid   = (string)array_shift($partes);
    $queue_name = (string)array_shift($partes);
    $agent      = (string)array_shift($partes);
    $event      = (string)array_shift($partes);
    if(count($partes)>0) { $data1 = array_shift($partes); } else { $data1=''; }
    if(count($partes)>0) { $data2 = array_shift($partes); } else { $data2=''; }
    if(count($partes)>0) { $data3 = array_shift($partes); } else { $data3=''; }

    if (preg_match('/[^0-9]/', $date)) {
        print "return preg match\n";
        return;
    }

    if($date == "" || $date < $last_event_ts) {
        return;
    }

    // strftime() is deprecated since PHP 8.1
    $date = date("Y-m-d H:i:s",(int)$date);
    $queue_id = check_queue($queue_name);
    $agent_id = check_agent($agent);

    if(array_key_exists($event,$event_array)) {
        $event_id = $event_array["$event"];
        if($agent_id <> -1) {
            $query = "INSERT IGNORE INTO queue_stats (uniqueid, datetime, qname, qagent, qevent, info1, info2, info3) ";
            $query.= "VALUES ('%s','%s','%s','%s','%s','%s','%s','%s')";
            $res = $midb->consulta($query,array($uniqueid,$date,$queue_id,$agent_id,$event_id,$data1,$data2,$data3));
        }
    }
}
?>
