<?php
require_once __DIR__.'/actions.php';

class commentActions extends appActions
{
	const COMMENTS_IN_PAGE = 20;

	public function executeComment()
	{
		$app_id = $this->app->getId();
		$comment_count = CommentDb::selectCountByAppId($app_id);

		$current_page = mfwRequest::param('page',1);
		$max_page = ceil($comment_count/self::COMMENTS_IN_PAGE);
		$offset = (max(0,min($current_page,$max_page)-1)) * self::COMMENTS_IN_PAGE;

		$comments = CommentDb::selectByAppId($app_id,self::COMMENTS_IN_PAGE,$offset);

		$commented_package = PackageDb::retrieveByPKs($comments->getColumnArray('package_id'));

		$install_packages = $this->login_user->getInstallPackages($app_id);
		$install_packages->sort(function($a,$b){ return $a['id'] < $b['id']; });

		$params = array(
			'comments_in_page' => self::COMMENTS_IN_PAGE,
			'comment_count' => $comment_count,
			'comments' => $comments,
			'commented_package' => $commented_package,
			'cur_page' => $current_page,
			'max_page' => $max_page,
			'install_packages' => $install_packages,
			);
		return $this->build($params);
	}

	public function executeComment_post()
	{
		var_dump($_POST);
	}

}

