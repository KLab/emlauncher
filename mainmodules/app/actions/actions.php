<?php
require_once APP_ROOT.'/model/Application.php';

class appActions extends MainActions
{
	protected $app = null;

	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}
		if(in_array($this->getAction(),array('new','create'))){
			return null;
		}
		$id = mfwRequest::param('id');
		$this->app = ApplicationDb::retrieveByPK($id);
		if(!$this->app){
			return $this->buildErrorPage('Not Found',array(self::HTTP_404_NOTFOUND));
		}
		return null;
	}

	public function build($params)
	{
		if(!isset($params['app'])){
			$params['app'] = $this->app;
		}
		return parent::build($params);
	}

	public function executeNew()
	{
		$params = array(
			);
		return $this->build($params);
	}

	public function executeCreate()
	{
		$title = mfwRequest::param('title');
		$data = mfwRequest::param('icon-data');
		$description = mfwRequest::param('description');
		if(!$title || !preg_match('/^data:[^;]+;base64,(.+)$/',$data,$match)){
			return $this->response(self::HTTP_400_BADREQUEST);
		}
		$image = base64_decode($match[1]);

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$app = ApplicationDb::insertNewApp(
				$this->login_user,$title,$image,$description);
			$con->commit();
		}
		catch(Exception $e){
			$con->rollback();
			throw $e;
		}

		return $this->redirect("/app?id={$app->getId()}");
	}

	public function executeIndex()
	{
		$platform = mfwRequest::param('pf');
		if(!in_array($platform,array('android','ios','all'))){
			$platform = 'android';// fixme: UA見て変える
		}

		$owners = $this->app->getOwners();
		$ownerid = $owners->searchPK('owner_mail',$this->login_user->getMail());

		$params = array(
			'pf' => $platform,
			'is_owner' => ($ownerid!==null),
			);
		return $this->build($params);
	}

}
