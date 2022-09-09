<?php
include 'config.php';
include 'inc.authorize.php';

$auth = new authorize(AUTHORIZED);
$auth->authorize();
echo file_get_contents(SOLPROXY_HOST."?action=server-status");