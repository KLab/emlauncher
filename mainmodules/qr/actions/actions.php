<?php
use Endroid\QrCode\QrCode;
use Endroid\QrCode\Encoding\Encoding;
use Endroid\QrCode\ErrorCorrectionLevel\ErrorCorrectionLevelHigh;
use Endroid\QrCode\Writer\PngWriter;

class qrActions extends MainActions
{
    const DEFAULT_QR_SIZE = 150;
    const DEFAULT_QR_PADDING_SIZE = 10;
    /**
     * EMLauncher用のQRコードを生成します。
     * 渡されるべき q は "/app?id=1"のようなQueryStringのみ
     * Action側でEMLauncherのHOSTを追加してQRコードを生成します
     *
     * @return array
     * @throws \Endroid\QrCode\Exceptions\ImageFunctionUnknownException
     */
    public function executeCode()
    {
        $query = mfwRequest::param('q', "");
        $url = mfwRequest::makeUrl($query);
        $size =(int)mfwRequest::param('s', 150);
        if ($size > 300) {
            $size = 300;
        }

        $writer = new PngWriter();
        $qr_code = QrCode::create($url)
            ->setEncoding(new Encoding('UTF-8'))
            ->setErrorCorrectionLevel(new ErrorCorrectionLevelHigh())
            ->setSize($size)
            ->setMargin(self::DEFAULT_QR_PADDING_SIZE);
        return $this->pngResponse(
            $writer->write($qr_code)->getString()
        );
    }

    private function pngResponse($contents)
    {
        $header = array(
            'Content-type: image/png',
        );
        return array($header,$contents);
    }

    private static function getRenderSize($size, $padding)
    {
        $size = $size - 2 * $padding;
        if ($size <= 0) {
            $size = self::DEFAULT_QR_SIZE;
        }
        return $size;
    }

}
