<?php
require_once __DIR__.'/../initialize.php';
require_once APP_ROOT.'/mainmodules/MainModules.php';

try{
	mfwServerEnv::setEnv($_SERVER['MFW_ENV']);

	list($headers,$content) = MainModules::execute();
	foreach($headers as $h){
		header($h);
	}
	echo $content;
}
catch(Exception $e){
	header(mfwActions::HTTP_500_INTERNALSERVERERROR);
	echo "<h1>500 Internal Server Error</h1>\n";
	echo $e->getMessage();
	apache_log('exception',$e->getMessage());
}

