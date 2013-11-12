<?php
require_once APP_ROOT.'/model/Package.php';

class packageActions extends MainActions
{
	protected $package = null;

	public function initialize()
	{
		if(($err=parent::initialize())){
			return $err;
		}
		$id = mfwRequest::param('id');
		$this->package = PackageDb::retrieveByPK($id);
		if(!$this->package){
			return $this->buildErrorPage('Not Found',array(self::HTTP_404_NOTFOUND));
		}
		return null;
	}

	public function build($params)
	{
		$params['package'] = $this->package;
		$params['app'] = $this->package->getApplication();
		return parent::build($params);
	}


	public function executeIndex()
	{
		$params = array(
			);
		return $this->build($params);
	}
}