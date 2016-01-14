<?php
require_once (APP_ROOT.'/libs/QrCode/src/QrCode.php');
use Endroid\QrCode\QrCode;

class qrActions extends MainActions
{
    public function executeCode()
    {
        $text = mfwRequest::param('q', "");
        $size =(int)mfwRequest::param('s', 150);
        if ($size > 300) {
            $size = 300;
        }
        $qr_code = new QrCode();
        $qr_code
            ->setText($text)
            ->setSize($size)
            ->setPadding(10)
            ->setErrorCorrection('high');
        return $this->pngResponse(
            $qr_code->get()
        );

    }

    private function pngResponse($contents)
    {
        $header = array(
            'Content-type: image/png',
        );
        return array($header,$contents);
    }

}
