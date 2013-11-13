<?php
require_once __DIR__.'/actions.php';

class editActions extends packageActions
{
	protected $package = null;
	protected $app = null;

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

	public function executeEdit()
	{
		$params = array(
			);
		return $this->build($params);
	}

	public function executeEdit_commit()
	{
		var_dump($_POST);
		return;

		$title = mfwRequest::param('title');
		$description = mfwRequest::param('description');
		$tag_names = mfwRequest::param('tags');

		return $this->redirect("/package?id={$this->package->getId()}");
	}

}

