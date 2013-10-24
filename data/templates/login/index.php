
<div id="signin-dialog">
 <div class="titlebox">
  <h1 class="title">Login</h1>
  <div class="content">
    <?php if($enable_password): ?>
    <form  method="post" action="<?=url('/login/post')?>">
      <dl>
        <dt>email</dt>
        <dd><input type="text"></dd>
        <dt>password</dt>
        <dd><input type="password"></dd>
        <dt></dt>
        <dd>
          <input type="submit" class="active-button" value="login">
          <a class="forget-pass" href="">forget password</a>
        </dd>
      </dl>
    </form>
    <?php endif ?>
    <?php if($enable_google_auth): ?>
    <?=($enable_password)?'<hr>':''?>
    <a class="google-login active-button" href="">
      login with google account
    </a>
    <?php endif ?>
  </div>
</div>
</div>