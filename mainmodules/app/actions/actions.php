<?php
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Package.php';

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
		$repository = mfwRequest::param('repository');
		if(!$title || !preg_match('/^data:[^;]+;base64,(.+)$/',$data,$match)){
			error_log(__METHOD__.": bad request: $title, ".substr($data,0,30));
			return $this->response(self::HTTP_400_BADREQUEST);
		}
		$image = base64_decode($match[1]);

		$con = mfwDBConnection::getPDO();
		$con->beginTransaction();
		try{
			$app = ApplicationDb::insertNewApp(
				$this->login_user,$title,$image,$description,$repository);
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
		static $pf = array(
			'android' => Package::PF_ANDROID,
			'ios' => Package::PF_IOS,
			'all' => null,
			);

		$platform = mfwRequest::param('pf');
		if(!in_array($platform,array('android','ios','all'))){
			$ua = mfwRequest::userAgent();
			if($ua->isAndroid()){
				$platform = 'android';
			}
			elseif($ua->isIOS()){
				$platform = 'ios';
			}
			else{
				$platform = 'all';
			}
		}

		$pkgs = PackageDb::selectByAppId($this->app->getId(),$pf[$platform]);
		$params = array(
			'pf' => $platform,
			'is_owner' => $this->app->isOwner($this->login_user),
			'packages' => $pkgs,
			);
		return $this->build($params);
	}

}
