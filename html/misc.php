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

function return_timestamp($date_string)
{
  $parts = array_pad(preg_split("/-|:| /",(string)$date_string,6), 6, 0);
  list ($year,$month,$day,$hour,$min,$sec) = array_map('intval', $parts);
  $u_timestamp = mktime($hour,$min,$sec,$month,$day,$year);
  return $u_timestamp;
}

// Escape a value for HTML output (text and attribute values).
function h($text)
{
  return htmlspecialchars((string)$text, ENT_QUOTES, 'UTF-8');
}

/*
   Bar chart drawn in the browser by js/charts.js (Chart.js).

   $labels  one label per bar (or per group of bars)
   $series  series name => list of values, one value per label
   $options 'stacked' => true  stack the series on top of each other
            'wide'    => true  full-width chart instead of a half-width one

   The data travels as JSON in a data attribute, so names with quotes or
   other special characters are safe.
*/
function chart_bar($id, $title, $labels, $series, $options = array()) {
    $datasets = array();
    foreach($series as $name=>$values) {
        $datasets[] = array('label'=>(string)$name, 'data'=>array_map('floatval', array_values($values)));
    }
    $config = array(
        'title'    => (string)$title,
        'labels'   => array_map('strval', array_values($labels)),
        'datasets' => $datasets,
        'stacked'  => !empty($options['stacked']),
    );
    $class = empty($options['wide']) ? 'chart' : 'chart chart-wide';
    echo "<div class='$class'><canvas id='".h($id)."' role='img' aria-label='".h($title)."' data-chart='".h(json_encode($config))."'></canvas></div>\n";
}

function tooltip($texto,$width) {
 echo " onmouseover=\"this.T_WIDTH=$width;this.T_PADDING=5;this.T_STICKY = false; return escape('$texto')\" ";
}


function print_exports($header_pdf,$data_pdf,$width_pdf,$title_pdf,$cover_pdf) {
		global $lang;
		global $language;
		$head_serial = serialize($header_pdf);
		$data_serial = serialize($data_pdf);
		$width_serial = serialize($width_pdf);
		$title_serial = serialize($title_pdf);
		$cover_serial = serialize($cover_pdf);
		$head_serial = rawurlencode($head_serial);
		$data_serial = rawurlencode($data_serial);
		$width_serial = rawurlencode($width_serial);
		$title_serial = rawurlencode($title_serial);
		$cover_serial = rawurlencode($cover_serial);
		echo "<BR><form method=post action='export.php'>\n";
		echo $lang["$language"]['export'];
		echo "<input type='hidden' name='head' value='".$head_serial."' />\n";
		echo "<input type='hidden' name='rawdata' value='".$data_serial."' />\n";
		echo "<input type='hidden' name='width' value='".$width_serial."' />\n";
		echo "<input type='hidden' name='title' value='".$title_serial."' />\n";
		echo "<input type='hidden' name='cover' value='".$cover_serial."' />\n";
		echo "<input type=image name='pdf' alt='export to pdf' src='images/pdf.gif' ";
		tooltip($lang["$language"]['pdfhelp'],200);
		echo ">\n";
		echo "<input type=image name='csv' alt='export to csv' src='images/excel.gif' "; 
		tooltip($lang["$language"]['csvhelp'],200);
		echo ">\n";
		echo "</form>";
}

// Turns a "'q1','q2'" list coming from the request/session into a list that
// is safe to embed in SQL. The query is later run through vsprintf(), which
// throws a ValueError on PHP 8 for stray % signs, so those are doubled too.
function sql_in_list($list) {
    global $midb;
    $items = array();
    foreach(explode(",", (string)$list) as $item) {
        $item = trim($item);
        if(strlen($item) >= 2 && $item[0] == "'" && substr($item, -1) == "'") {
            $item = substr($item, 1, -1);
        }
        $items[] = "'".str_replace('%', '%%', $midb->escape_string($item))."'";
    }
    return implode(",", $items);
}

function seconds2minutes($segundos) {
    // Implicit float to int conversion in % is deprecated since PHP 8.1
    $segundos = (int)$segundos;
    $minutos = intval($segundos / 60);
    $segundos = $segundos % 60;
    if(strlen($segundos)==1) {
		$segundos = "0".$segundos;
	}
    return "$minutos:$segundos";
}
?>
