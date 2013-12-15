<?php
require_once APP_ROOT.'/model/Application.php';
require_once APP_ROOT.'/model/Paging.php';

class topActions extends MainActions
{
	public function executeIndex()
	{
                $current_page = mfwRequest::param('page', 1, 'GET');
                $paging = new Paging($current_page, ApplicationDb::getRecordCount());
                if (!is_numeric($current_page) || $current_page <= 0 || $current_page > $paging->getTotalPageNumber()) {
                        $current_page = 1;
                }
		$apps = ApplicationDb::selectAllByUpdateOrderWithLimit(
                        $paging->getPageStartOffset($current_page, Paging::LINE_IN_PAGE)
                        );
		$params = array(
			'applications' => $apps,
                        'paging' => $paging,
			);
		return $this->build($params);
	}
}