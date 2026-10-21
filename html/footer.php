<?php
/*
   Copyright 2026 Krzysztof Hajdamowicz
   Part of Asternic Call Center Stats Lite, GPL-3.0-or-later (see LICENSE).

   Common page footer, the counterpart of header.php.
*/
$year = date('Y');
?>
</main>
<footer class="site-footer">
&copy; 2007&ndash;2026 Nicol&aacute;s Gudi&ntilde;o (<a href='https://www.asternic.net'>Asternic</a>) &middot;
&copy; 2026<?php if($year > 2026) echo "&ndash;$year"; ?> Krzysztof Hajdamowicz &middot;
<a href='https://github.com/KrzysztofHajdamowicz/asternic-stats-lite'>source</a> &middot;
licensed under <a href='https://www.gnu.org/licenses/gpl-3.0.html'>GPL-3.0</a>
</footer>
</body>
</html>
