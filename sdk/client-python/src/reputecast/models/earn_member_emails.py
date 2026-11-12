from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.earn_member_email import EarnMemberEmail





T = TypeVar("T", bound="EarnMemberEmails")



@_attrs_define
class EarnMemberEmails:
    """ 
        Attributes:
            can_publish (bool):
            category_kind (str):
            category_name (str):
            emails (list[EarnMemberEmail]):
            emails_url (str):
            from_email (str):
            from_name (str):
            helpers (list[str]):
            reply_to (str):
            route_fallback (str):
            route_primary (str):
            tier_short (str):
            first_gift_ending (datetime.datetime | Unset):
     """

    can_publish: bool
    category_kind: str
    category_name: str
    emails: list[EarnMemberEmail]
    emails_url: str
    from_email: str
    from_name: str
    helpers: list[str]
    reply_to: str
    route_fallback: str
    route_primary: str
    tier_short: str
    first_gift_ending: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_member_email import EarnMemberEmail # noqa: PLC0415
        can_publish = self.can_publish

        category_kind = self.category_kind

        category_name = self.category_name

        emails = []
        for emails_item_data in self.emails:
            emails_item = emails_item_data.to_dict()
            emails.append(emails_item)



        emails_url = self.emails_url

        from_email = self.from_email

        from_name = self.from_name

        helpers = self.helpers



        reply_to = self.reply_to

        route_fallback = self.route_fallback

        route_primary = self.route_primary

        tier_short = self.tier_short

        first_gift_ending: str | Unset = UNSET
        if not isinstance(self.first_gift_ending, Unset):
            first_gift_ending = self.first_gift_ending.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canPublish": can_publish,
            "categoryKind": category_kind,
            "categoryName": category_name,
            "emails": emails,
            "emailsUrl": emails_url,
            "fromEmail": from_email,
            "fromName": from_name,
            "helpers": helpers,
            "replyTo": reply_to,
            "routeFallback": route_fallback,
            "routePrimary": route_primary,
            "tierShort": tier_short,
        })
        if first_gift_ending is not UNSET:
            field_dict["firstGiftEnding"] = first_gift_ending

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_member_email import EarnMemberEmail # noqa: PLC0415
        d = dict(src_dict)
        can_publish = d.pop("canPublish")

        category_kind = d.pop("categoryKind")

        category_name = d.pop("categoryName")

        emails = []
        _emails = d.pop("emails")
        for emails_item_data in (_emails):
            emails_item = EarnMemberEmail.from_dict(emails_item_data)



            emails.append(emails_item)


        emails_url = d.pop("emailsUrl")

        from_email = d.pop("fromEmail")

        from_name = d.pop("fromName")

        helpers = cast(list[str], d.pop("helpers"))


        reply_to = d.pop("replyTo")

        route_fallback = d.pop("routeFallback")

        route_primary = d.pop("routePrimary")

        tier_short = d.pop("tierShort")

        _first_gift_ending = d.pop("firstGiftEnding", UNSET)
        first_gift_ending: datetime.datetime | Unset
        if isinstance(_first_gift_ending,  Unset):
            first_gift_ending = UNSET
        else:
            first_gift_ending = datetime.datetime.fromisoformat(_first_gift_ending)




        earn_member_emails = cls(
            can_publish=can_publish,
            category_kind=category_kind,
            category_name=category_name,
            emails=emails,
            emails_url=emails_url,
            from_email=from_email,
            from_name=from_name,
            helpers=helpers,
            reply_to=reply_to,
            route_fallback=route_fallback,
            route_primary=route_primary,
            tier_short=tier_short,
            first_gift_ending=first_gift_ending,
        )

        return earn_member_emails

