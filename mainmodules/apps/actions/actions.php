<?php
require_once APP_ROOT.'/model/Application.php';

class appsActions extends MainActions
{
	public function executeIndex()
	{
		//仮
		$url = mfwRequest::makeUrl('/apps/new');
		return array(array(),"<a href=\"$url\">new</a>");
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

		return $this->redirect("/apps/detail?id={$app->getId()}");
	}

}
