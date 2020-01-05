
import jQuery from 'jquery';

import 'bootstrap';
import 'bootstrap/dist/css/bootstrap.css';

import '../css/main.css';
import '../css/util.css';

(function ($) {
  "use strict";

  window.tryAnswer = function (_this) {

    var bt = $(_this);
    var bg = $("#q-form").find(".contact100-form-bgbtn");
    var txt = $("#q-form").find(".contact100-form-title");
    var name = $("#q-form").find("input[name='name']");
    var answer = $("#q-form").find("input[name='answer']");

    if (!name.val().trim().length) {
      name.parent().addClass("alert-validate");
      return false;
    }
    if (!answer.val().trim().length) {
      answer.parent().addClass("alert-validate");
      return false;
    }

    name.parent().removeClass("alert-validate");
    answer.parent().removeClass("alert-validate");

    bt.attr("disabled", true);
    bg.css("background", "#9e9e9e");

    var winner = function (approval) {
      var q = $("#q");
      var a = $("#a");
      var msg = a.find(".wrap-contact150");

      q.addClass("hidden");
      a.removeClass("hidden");

      if (!approval.length) {
        return msg.text("ну что-то сломалось 😅");
      }

      msg.empty();

      var lines = ['alert-primary', 'alert-secondary', 'alert-success', 'alert-danger'];
      for (var i = 0; i < approval.length; i++) {
        msg.append($('<div class="alert ' + lines[i] + '">' + approval[i] + '</div>'));
      }
    }

    var notpass = function () {
      bt.attr("disabled", false);
      bg.css("background", "");
      txt.text("хм, давай еще 🧐");
      setTimeout((bt.addClass("animated shake"), function () { bt.removeClass("animated shake") }), 1000);
    }

    var rocking = setInterval((txt.text("хм 🚀"), function () { txt.text(txt.text() + "🚀") }), 50);

    fetch('//127.0.0.1:8081/answer/check', {
      method: 'POST',
      body: JSON.stringify({ answer: answer.val().trim(), name: name.val().trim() }),
    })
      .then(function (data) { return data.json() })
      .catch(function () { })
      .then(function (approval) {
        clearInterval(rocking);
        if (approval) {
          winner(approval);
        } else {
          notpass();
        }
      })

    return false;
  }

})(jQuery);