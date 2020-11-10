<?php
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Package.php';
require_once APP_ROOT.'/model/Comment.php';

class appActions extends MainActions
{
	protected $app = null;

	const LINE_IN_PAGE = 20;

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

	public function build($params=array(),$headers=array())
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
			error_log(__METHOD__.'('.__LINE__."): bad request: $title, ".substr($data,0,30));
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
			error_log(__METHOD__.'('.__LINE__.'): '.get_class($e).":{$e->getMessage()}");
			$con->rollback();
			throw $e;
		}

		apache_log('app_id',$app->getId());

		return $this->redirect("/app?id={$app->getId()}");
	}

	public function executeIndex()
	{
		static $pf = array(
			'android' => Package::PF_ANDROID,
			'ios' => Package::PF_IOS,
			'other' => Package::PF_UNKNOWN,
			'all' => null,
			);

		$platform = mfwRequest::param('pf');
		if(!in_array($platform,array('android','ios','other','all'))){
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

		$tags = mfwRequest::param('tags') ? explode(' ', mfwRequest::param('tags')) : array();
		$current_page = (int)mfwRequest::param('page', 1);
		if($current_page<1){
			$current_page = 1;
		}

		$offset = ($current_page - 1) * self::LINE_IN_PAGE;
		$pkgs = PackageDb::selectByAppIdPfTagsWithLimit(
			$this->app->getId(), $pf[$platform], $tags, $offset, self::LINE_IN_PAGE + 1
		);

		$has_next_page = false;
		if ($pkgs->count() > self::LINE_IN_PAGE) {
			$pkgs = $pkgs->slice(0, self::LINE_IN_PAGE);
			$has_next_page = true;
		}

		$comment_count = CommentDb::selectCountByAppId($this->app->getId());
		$top_comments = CommentDb::selectByAppId($this->app->getId(),2);

		$commented_package = PackageDb::retrieveByPKs($top_comments->getColumnArray('package_id'));

		$params = array(
			'pf' => $platform,
			'is_owner' => $this->app->isOwner($this->login_user),
			'packages' => $pkgs,
			'active_tags' => $tags,
			'current_page' => $current_page,
			'has_next_page' => $has_next_page,
			'filter_open' => mfwRequest::param('filter_open'),
			'top_comments' => $top_comments,
			'comment_count' => $comment_count,
			'commented_package' => $commented_package,
		);
		return $this->build($params);
	}

}
