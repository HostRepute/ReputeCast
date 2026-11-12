from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_contact_list_email import SmsContactListEmail
from ..models.sms_contact_list_sms import SmsContactListSms
from ..models.sms_contact_list_whats_app import SmsContactListWhatsApp
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsContactList")



@_attrs_define
class SmsContactList:
    """ 
        Attributes:
            email (SmsContactListEmail):
            list_id (str):
            name (str):
            sms (SmsContactListSms):
            whats_app (SmsContactListWhatsApp):
            confirmation (str | Unset):
            confirmation_at (datetime.datetime | Unset):
            email_at (datetime.datetime | Unset):
            email_confirmed (bool | Unset):
            email_source (str | Unset):
            email_source_label (str | Unset):
            sms_at (datetime.datetime | Unset):
            sms_ip (str | Unset):
            sms_keyword (str | Unset):
            sms_page (str | Unset):
            sms_source (str | Unset):
            sms_version (str | Unset):
            whats_app_at (datetime.datetime | Unset):
            whats_app_keyword (str | Unset):
            whats_app_page (str | Unset):
            whats_app_source (str | Unset):
            whats_app_version (str | Unset):
     """

    email: SmsContactListEmail
    list_id: str
    name: str
    sms: SmsContactListSms
    whats_app: SmsContactListWhatsApp
    confirmation: str | Unset = UNSET
    confirmation_at: datetime.datetime | Unset = UNSET
    email_at: datetime.datetime | Unset = UNSET
    email_confirmed: bool | Unset = UNSET
    email_source: str | Unset = UNSET
    email_source_label: str | Unset = UNSET
    sms_at: datetime.datetime | Unset = UNSET
    sms_ip: str | Unset = UNSET
    sms_keyword: str | Unset = UNSET
    sms_page: str | Unset = UNSET
    sms_source: str | Unset = UNSET
    sms_version: str | Unset = UNSET
    whats_app_at: datetime.datetime | Unset = UNSET
    whats_app_keyword: str | Unset = UNSET
    whats_app_page: str | Unset = UNSET
    whats_app_source: str | Unset = UNSET
    whats_app_version: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        email = self.email.value

        list_id = self.list_id

        name = self.name

        sms = self.sms.value

        whats_app = self.whats_app.value

        confirmation = self.confirmation

        confirmation_at: str | Unset = UNSET
        if not isinstance(self.confirmation_at, Unset):
            confirmation_at = self.confirmation_at.isoformat()

        email_at: str | Unset = UNSET
        if not isinstance(self.email_at, Unset):
            email_at = self.email_at.isoformat()

        email_confirmed = self.email_confirmed

        email_source = self.email_source

        email_source_label = self.email_source_label

        sms_at: str | Unset = UNSET
        if not isinstance(self.sms_at, Unset):
            sms_at = self.sms_at.isoformat()

        sms_ip = self.sms_ip

        sms_keyword = self.sms_keyword

        sms_page = self.sms_page

        sms_source = self.sms_source

        sms_version = self.sms_version

        whats_app_at: str | Unset = UNSET
        if not isinstance(self.whats_app_at, Unset):
            whats_app_at = self.whats_app_at.isoformat()

        whats_app_keyword = self.whats_app_keyword

        whats_app_page = self.whats_app_page

        whats_app_source = self.whats_app_source

        whats_app_version = self.whats_app_version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "listId": list_id,
            "name": name,
            "sms": sms,
            "whatsApp": whats_app,
        })
        if confirmation is not UNSET:
            field_dict["confirmation"] = confirmation
        if confirmation_at is not UNSET:
            field_dict["confirmationAt"] = confirmation_at
        if email_at is not UNSET:
            field_dict["emailAt"] = email_at
        if email_confirmed is not UNSET:
            field_dict["emailConfirmed"] = email_confirmed
        if email_source is not UNSET:
            field_dict["emailSource"] = email_source
        if email_source_label is not UNSET:
            field_dict["emailSourceLabel"] = email_source_label
        if sms_at is not UNSET:
            field_dict["smsAt"] = sms_at
        if sms_ip is not UNSET:
            field_dict["smsIp"] = sms_ip
        if sms_keyword is not UNSET:
            field_dict["smsKeyword"] = sms_keyword
        if sms_page is not UNSET:
            field_dict["smsPage"] = sms_page
        if sms_source is not UNSET:
            field_dict["smsSource"] = sms_source
        if sms_version is not UNSET:
            field_dict["smsVersion"] = sms_version
        if whats_app_at is not UNSET:
            field_dict["whatsAppAt"] = whats_app_at
        if whats_app_keyword is not UNSET:
            field_dict["whatsAppKeyword"] = whats_app_keyword
        if whats_app_page is not UNSET:
            field_dict["whatsAppPage"] = whats_app_page
        if whats_app_source is not UNSET:
            field_dict["whatsAppSource"] = whats_app_source
        if whats_app_version is not UNSET:
            field_dict["whatsAppVersion"] = whats_app_version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = SmsContactListEmail(d.pop("email"))




        list_id = d.pop("listId")

        name = d.pop("name")

        sms = SmsContactListSms(d.pop("sms"))




        whats_app = SmsContactListWhatsApp(d.pop("whatsApp"))




        confirmation = d.pop("confirmation", UNSET)

        _confirmation_at = d.pop("confirmationAt", UNSET)
        confirmation_at: datetime.datetime | Unset
        if isinstance(_confirmation_at,  Unset):
            confirmation_at = UNSET
        else:
            confirmation_at = datetime.datetime.fromisoformat(_confirmation_at)




        _email_at = d.pop("emailAt", UNSET)
        email_at: datetime.datetime | Unset
        if isinstance(_email_at,  Unset):
            email_at = UNSET
        else:
            email_at = datetime.datetime.fromisoformat(_email_at)




        email_confirmed = d.pop("emailConfirmed", UNSET)

        email_source = d.pop("emailSource", UNSET)

        email_source_label = d.pop("emailSourceLabel", UNSET)

        _sms_at = d.pop("smsAt", UNSET)
        sms_at: datetime.datetime | Unset
        if isinstance(_sms_at,  Unset):
            sms_at = UNSET
        else:
            sms_at = datetime.datetime.fromisoformat(_sms_at)




        sms_ip = d.pop("smsIp", UNSET)

        sms_keyword = d.pop("smsKeyword", UNSET)

        sms_page = d.pop("smsPage", UNSET)

        sms_source = d.pop("smsSource", UNSET)

        sms_version = d.pop("smsVersion", UNSET)

        _whats_app_at = d.pop("whatsAppAt", UNSET)
        whats_app_at: datetime.datetime | Unset
        if isinstance(_whats_app_at,  Unset):
            whats_app_at = UNSET
        else:
            whats_app_at = datetime.datetime.fromisoformat(_whats_app_at)




        whats_app_keyword = d.pop("whatsAppKeyword", UNSET)

        whats_app_page = d.pop("whatsAppPage", UNSET)

        whats_app_source = d.pop("whatsAppSource", UNSET)

        whats_app_version = d.pop("whatsAppVersion", UNSET)

        sms_contact_list = cls(
            email=email,
            list_id=list_id,
            name=name,
            sms=sms,
            whats_app=whats_app,
            confirmation=confirmation,
            confirmation_at=confirmation_at,
            email_at=email_at,
            email_confirmed=email_confirmed,
            email_source=email_source,
            email_source_label=email_source_label,
            sms_at=sms_at,
            sms_ip=sms_ip,
            sms_keyword=sms_keyword,
            sms_page=sms_page,
            sms_source=sms_source,
            sms_version=sms_version,
            whats_app_at=whats_app_at,
            whats_app_keyword=whats_app_keyword,
            whats_app_page=whats_app_page,
            whats_app_source=whats_app_source,
            whats_app_version=whats_app_version,
        )

        return sms_contact_list

