<?php
require_once __DIR__.'/actions.php';

class preferenceActions extends appActions
{
	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}
		if(!$this->app->isOwner($this->login_user)){
			return $this->buildErrorPage(
				'Permission Denied',array(self::HTTP_403_FORBIDDEN));
		}
		return null;
	}

	public function executePreference()
	{
		$params = array(
			);
		return $this->build($params);
	}

	public function executePreference_refresh_apikey()
	{
		$oldkey = mfwRequest::param('api_key','POST');
		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$this->app = ApplicationDb::retrieveByPkForUpdate($this->app->getId());
			if($this->app->getApiKey()===$oldkey){
				$this->app->refreshApiKey($con);
			}
			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			error_log(__METHOD__.": {$e->getMessage()}");
			throw $e;
		}
		return $this->redirect("/app/preference?id={$this->app->getId()}");
	}


}
