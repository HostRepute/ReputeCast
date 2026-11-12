from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.whats_app_business import WhatsAppBusiness
  from ..models.whats_app_contact_list import WhatsAppContactList
  from ..models.whats_app_suppression import WhatsAppSuppression
  from ..models.whats_app_thread_item import WhatsAppThreadItem
  from ..models.whats_app_window import WhatsAppWindow





T = TypeVar("T", bound="WhatsAppContactCard")



@_attrs_define
class WhatsAppContactCard:
    """ 
        Attributes:
            can_edit (bool):
            can_reply (bool):
            connected (bool):
            country (str):
            display (str):
            email (str):
            line_type (str):
            lists (list[WhatsAppContactList]):
            name (str):
            phone (str):
            thread (list[WhatsAppThreadItem]):
            time_zone (str):
            window (WhatsAppWindow):
            business (WhatsAppBusiness | Unset):
            suppressed (WhatsAppSuppression | Unset):
     """

    can_edit: bool
    can_reply: bool
    connected: bool
    country: str
    display: str
    email: str
    line_type: str
    lists: list[WhatsAppContactList]
    name: str
    phone: str
    thread: list[WhatsAppThreadItem]
    time_zone: str
    window: WhatsAppWindow
    business: WhatsAppBusiness | Unset = UNSET
    suppressed: WhatsAppSuppression | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_business import WhatsAppBusiness # noqa: PLC0415
        from ..models.whats_app_contact_list import WhatsAppContactList # noqa: PLC0415
        from ..models.whats_app_suppression import WhatsAppSuppression # noqa: PLC0415
        from ..models.whats_app_thread_item import WhatsAppThreadItem # noqa: PLC0415
        from ..models.whats_app_window import WhatsAppWindow # noqa: PLC0415
        can_edit = self.can_edit

        can_reply = self.can_reply

        connected = self.connected

        country = self.country

        display = self.display

        email = self.email

        line_type = self.line_type

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        name = self.name

        phone = self.phone

        thread = []
        for thread_item_data in self.thread:
            thread_item = thread_item_data.to_dict()
            thread.append(thread_item)



        time_zone = self.time_zone

        window = self.window.to_dict()

        business: dict[str, Any] | Unset = UNSET
        if not isinstance(self.business, Unset):
            business = self.business.to_dict()

        suppressed: dict[str, Any] | Unset = UNSET
        if not isinstance(self.suppressed, Unset):
            suppressed = self.suppressed.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "canReply": can_reply,
            "connected": connected,
            "country": country,
            "display": display,
            "email": email,
            "lineType": line_type,
            "lists": lists,
            "name": name,
            "phone": phone,
            "thread": thread,
            "timeZone": time_zone,
            "window": window,
        })
        if business is not UNSET:
            field_dict["business"] = business
        if suppressed is not UNSET:
            field_dict["suppressed"] = suppressed

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_business import WhatsAppBusiness # noqa: PLC0415
        from ..models.whats_app_contact_list import WhatsAppContactList # noqa: PLC0415
        from ..models.whats_app_suppression import WhatsAppSuppression # noqa: PLC0415
        from ..models.whats_app_thread_item import WhatsAppThreadItem # noqa: PLC0415
        from ..models.whats_app_window import WhatsAppWindow # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        can_reply = d.pop("canReply")

        connected = d.pop("connected")

        country = d.pop("country")

        display = d.pop("display")

        email = d.pop("email")

        line_type = d.pop("lineType")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = WhatsAppContactList.from_dict(lists_item_data)



            lists.append(lists_item)


        name = d.pop("name")

        phone = d.pop("phone")

        thread = []
        _thread = d.pop("thread")
        for thread_item_data in (_thread):
            thread_item = WhatsAppThreadItem.from_dict(thread_item_data)



            thread.append(thread_item)


        time_zone = d.pop("timeZone")

        window = WhatsAppWindow.from_dict(d.pop("window"))




        _business = d.pop("business", UNSET)
        business: WhatsAppBusiness | Unset
        if isinstance(_business,  Unset):
            business = UNSET
        else:
            business = WhatsAppBusiness.from_dict(_business)




        _suppressed = d.pop("suppressed", UNSET)
        suppressed: WhatsAppSuppression | Unset
        if isinstance(_suppressed,  Unset):
            suppressed = UNSET
        else:
            suppressed = WhatsAppSuppression.from_dict(_suppressed)




        whats_app_contact_card = cls(
            can_edit=can_edit,
            can_reply=can_reply,
            connected=connected,
            country=country,
            display=display,
            email=email,
            line_type=line_type,
            lists=lists,
            name=name,
            phone=phone,
            thread=thread,
            time_zone=time_zone,
            window=window,
            business=business,
            suppressed=suppressed,
        )

        return whats_app_contact_card

