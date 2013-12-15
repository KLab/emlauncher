<?php

/**
 * Row object for 'application' table.
 */
class Paging {
	const LINE_IN_PAGE = 10;
        const CANDIDATE_PAGES_TO_BROWSE = 5;

	protected $current_page_number;
	protected $total_page_number;
        
        function __construct(int $current_page_number = 1, int $total_item_number = 1) {
                $this->current_page_number = $current_page_number;
                $this->total_page_number = (int)(max($total_item_number, 1) / self::LINE_IN_PAGE) + 1;
        }
	public function getCurrentPage(){
		return $this->current_page_number;
	}
	public function getTotalPageNumber(){
		return $this->total_page_number;
	}
        public function getPageStartOffset(int $page){
                return (max($page, 1) - 1) * self::LINE_IN_PAGE + 1;
        }
        public function setCurrentPage(int $current_page_number){
		$this->current_page_number = min(max($current_page_number, 1), $this->total_page_number);
	}
        public function setTotalPageNumber(int $total_item_number) {
                $this->total_page_number = (int)(max($total_item_number, 1) / self::LINE_IN_PAGE) + 1;
        }
}