from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SignupFormContent")



@_attrs_define
class SignupFormContent:
    """ 
        Attributes:
            button (str):
            description (str):
            email_placeholder (str):
            footnote (str):
            heading (str):
            lists_label (str):
            name_placeholder (str):
            popup_delay (int): Seconds before a popup opens
            popup_repeat_days (int): Days a closed popup stays closed
            popup_scroll (int): Percent of the page scrolled before a popup opens; 0 is off
            after_submit_text (str | Unset):
            hide_subscribed (bool | Unset):
            popup_exit (bool | Unset):
            sms_consent (bool | Unset):
            sms_consent_text (str | Unset):
            sms_consent_version (int | Unset):
     """

    button: str
    description: str
    email_placeholder: str
    footnote: str
    heading: str
    lists_label: str
    name_placeholder: str
    popup_delay: int
    popup_repeat_days: int
    popup_scroll: int
    after_submit_text: str | Unset = UNSET
    hide_subscribed: bool | Unset = UNSET
    popup_exit: bool | Unset = UNSET
    sms_consent: bool | Unset = UNSET
    sms_consent_text: str | Unset = UNSET
    sms_consent_version: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        button = self.button

        description = self.description

        email_placeholder = self.email_placeholder

        footnote = self.footnote

        heading = self.heading

        lists_label = self.lists_label

        name_placeholder = self.name_placeholder

        popup_delay = self.popup_delay

        popup_repeat_days = self.popup_repeat_days

        popup_scroll = self.popup_scroll

        after_submit_text = self.after_submit_text

        hide_subscribed = self.hide_subscribed

        popup_exit = self.popup_exit

        sms_consent = self.sms_consent

        sms_consent_text = self.sms_consent_text

        sms_consent_version = self.sms_consent_version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "button": button,
            "description": description,
            "emailPlaceholder": email_placeholder,
            "footnote": footnote,
            "heading": heading,
            "listsLabel": lists_label,
            "namePlaceholder": name_placeholder,
            "popupDelay": popup_delay,
            "popupRepeatDays": popup_repeat_days,
            "popupScroll": popup_scroll,
        })
        if after_submit_text is not UNSET:
            field_dict["afterSubmitText"] = after_submit_text
        if hide_subscribed is not UNSET:
            field_dict["hideSubscribed"] = hide_subscribed
        if popup_exit is not UNSET:
            field_dict["popupExit"] = popup_exit
        if sms_consent is not UNSET:
            field_dict["smsConsent"] = sms_consent
        if sms_consent_text is not UNSET:
            field_dict["smsConsentText"] = sms_consent_text
        if sms_consent_version is not UNSET:
            field_dict["smsConsentVersion"] = sms_consent_version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        button = d.pop("button")

        description = d.pop("description")

        email_placeholder = d.pop("emailPlaceholder")

        footnote = d.pop("footnote")

        heading = d.pop("heading")

        lists_label = d.pop("listsLabel")

        name_placeholder = d.pop("namePlaceholder")

        popup_delay = d.pop("popupDelay")

        popup_repeat_days = d.pop("popupRepeatDays")

        popup_scroll = d.pop("popupScroll")

        after_submit_text = d.pop("afterSubmitText", UNSET)

        hide_subscribed = d.pop("hideSubscribed", UNSET)

        popup_exit = d.pop("popupExit", UNSET)

        sms_consent = d.pop("smsConsent", UNSET)

        sms_consent_text = d.pop("smsConsentText", UNSET)

        sms_consent_version = d.pop("smsConsentVersion", UNSET)

        signup_form_content = cls(
            button=button,
            description=description,
            email_placeholder=email_placeholder,
            footnote=footnote,
            heading=heading,
            lists_label=lists_label,
            name_placeholder=name_placeholder,
            popup_delay=popup_delay,
            popup_repeat_days=popup_repeat_days,
            popup_scroll=popup_scroll,
            after_submit_text=after_submit_text,
            hide_subscribed=hide_subscribed,
            popup_exit=popup_exit,
            sms_consent=sms_consent,
            sms_consent_text=sms_consent_text,
            sms_consent_version=sms_consent_version,
        )

        return signup_form_content

