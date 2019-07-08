#!/usr/bin/php
<?php
/**
 * 古いパッケージを自動削除する.
 * cronに登録して使うなど.
 */
require_once __DIR__.'/../initialize.php';
require_once APP_ROOT.'/mfw/vendor/optionparse.php';
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Package.php';

$parser = new Optionparse(array(
		'name' => 'php delete_old_packages.php',
		'version' => '0.1',
		'description' => 'EMLauncher delete old packages',
		));

$parser->addOption('env',array(
		'short_name' => '-e',
		'long_name' => '--env',
		'description' => 'server environment. (default: local)',
		'help_name' => 'env',
		'default' => 'local',
		));

$parser->addOption('keep',array(
		'short_name' => '-k',
		'long_name' => '--keep',
		'description' => 'keeping package count (default: 100)',
		'help_name' => 'count',
		'default' => 100,
		));

$parser->addOption('dryrun',array(
		'short_name' => '-n',
		'long_name' => '--dryrun',
		'description' => "dryrun. don't delete files and records.",
		));

$parser->addOption('help',array(
		'short_name' => '-h',
		'long_name' => '--help',
		'description' => 'show this message.',
		));

$options = $parser->parse();

if($options['help']){
	$parser->displayUsage();
	exit(0);
}

$env = $options['env'];
$keep = $options['keep'];
$dryrun = $options['dryrun'];

// -----------------------------------------------------------------------------

echo "[START] ($env, keep=$keep) ".date('Y-m-d H:i:s')."\n";

mfwServerEnv::setEnv($env);
if($dryrun) echo "--DRYRUN--\n";

$apps = ApplicationDb::selectAll();

foreach($apps as $app){
    $packages = PackageDb::selectDeletablePackages($app,$keep);

	echo "Application {$app->getId()}: {$packages->count()} packages\n";

	foreach($packages as $pkg){

		echo "  delete package {$pkg->getId()} ";

		// パッケージ単位でトランザクションを組み、S3からも一つ一つ消していく
		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			// 削除対象より新しいpackageが必ず残っているはずなので、
			// applicationのアップデート時刻の更新などはしない.
			// packageとtagを操作するのでapplicationでロックする
			echo ".";
			$a = ApplicationDb::retrieveByPKForUpdate($app->getId());
			echo ".";
			$p = PackageDb::retrieveByPKForUpdate($pkg->getId());
			echo ".";
			if($p){
				if(!$dryrun) $p->delete();
				echo ".";
				if(!$dryrun) $p->deleteFile();
				echo ".";
				if(!$dryrun) $p->getAttachedFiles()->deleteFiles();
				echo ".";
			}
			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			error_log("Error on pakage {$pkg->getId()}: {$e->getMessage()}");
			throw $e;
		}

		echo " OK\n";
	}
}

echo "[END] ".date('Y-m-d H:i:s')."\n";

