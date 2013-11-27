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

}