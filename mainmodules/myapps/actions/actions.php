<?php

class myappsActions extends MainActions
{

	public function executeInstalled()
	{
		$installed_apps = InstallLog::getInstallApps($this->login_user);
		$installed_apps->sortByDesc('app_id');
		$params = array(
			'installed_apps' => $installed_apps,
			);
		return $this->build($params);
	}

	public function executeDelete()
	{
		$appid = mfwRequest::param('id');
		$instapp = InstallLog::getInstallApp($this->login_user,$appid);

		if($instapp){
			$instapp->delete();
		}

		return $this->redirect('/myapps/installed');
	}

	public function executeOwn()
	{
		$own_apps = ApplicationDb::selectOwnApps($this->login_user);
		$params = array(
			'own_apps' => $own_apps,
			);
		return $this->build($params);
	}

}