from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.website_access_list import WebsiteAccessList





T = TypeVar("T", bound="WebsiteMembersFacts")



@_attrs_define
class WebsiteMembersFacts:
    """ 
        Attributes:
            last_week (int):
            link_minutes (int):
            paid (int):
            session_days (int):
            signed_in (int):
            signin_on (bool):
            signup_lists (list[WebsiteAccessList]):
            signup_on (bool):
     """

    last_week: int
    link_minutes: int
    paid: int
    session_days: int
    signed_in: int
    signin_on: bool
    signup_lists: list[WebsiteAccessList]
    signup_on: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_access_list import WebsiteAccessList # noqa: PLC0415
        last_week = self.last_week

        link_minutes = self.link_minutes

        paid = self.paid

        session_days = self.session_days

        signed_in = self.signed_in

        signin_on = self.signin_on

        signup_lists = []
        for signup_lists_item_data in self.signup_lists:
            signup_lists_item = signup_lists_item_data.to_dict()
            signup_lists.append(signup_lists_item)



        signup_on = self.signup_on


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "lastWeek": last_week,
            "linkMinutes": link_minutes,
            "paid": paid,
            "sessionDays": session_days,
            "signedIn": signed_in,
            "signinOn": signin_on,
            "signupLists": signup_lists,
            "signupOn": signup_on,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_access_list import WebsiteAccessList # noqa: PLC0415
        d = dict(src_dict)
        last_week = d.pop("lastWeek")

        link_minutes = d.pop("linkMinutes")

        paid = d.pop("paid")

        session_days = d.pop("sessionDays")

        signed_in = d.pop("signedIn")

        signin_on = d.pop("signinOn")

        signup_lists = []
        _signup_lists = d.pop("signupLists")
        for signup_lists_item_data in (_signup_lists):
            signup_lists_item = WebsiteAccessList.from_dict(signup_lists_item_data)



            signup_lists.append(signup_lists_item)


        signup_on = d.pop("signupOn")

        website_members_facts = cls(
            last_week=last_week,
            link_minutes=link_minutes,
            paid=paid,
            session_days=session_days,
            signed_in=signed_in,
            signin_on=signin_on,
            signup_lists=signup_lists,
            signup_on=signup_on,
        )

        return website_members_facts

