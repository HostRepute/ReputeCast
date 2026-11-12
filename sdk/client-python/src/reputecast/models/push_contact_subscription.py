from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_contact_subscription_browser import PushContactSubscriptionBrowser
from ..models.push_contact_subscription_linked_by import PushContactSubscriptionLinkedBy
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PushContactSubscription")



@_attrs_define
class PushContactSubscription:
    """ 
        Attributes:
            browser (PushContactSubscriptionBrowser):
            id (str):
            linked_by (PushContactSubscriptionLinkedBy):
            platform (str):
            since (datetime.datetime):
            site (str):
            time_zone (str):
            last_seen (datetime.datetime | Unset):
     """

    browser: PushContactSubscriptionBrowser
    id: str
    linked_by: PushContactSubscriptionLinkedBy
    platform: str
    since: datetime.datetime
    site: str
    time_zone: str
    last_seen: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        browser = self.browser.value

        id = self.id

        linked_by = self.linked_by.value

        platform = self.platform

        since = self.since.isoformat()

        site = self.site

        time_zone = self.time_zone

        last_seen: str | Unset = UNSET
        if not isinstance(self.last_seen, Unset):
            last_seen = self.last_seen.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "browser": browser,
            "id": id,
            "linkedBy": linked_by,
            "platform": platform,
            "since": since,
            "site": site,
            "timeZone": time_zone,
        })
        if last_seen is not UNSET:
            field_dict["lastSeen"] = last_seen

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        browser = PushContactSubscriptionBrowser(d.pop("browser"))




        id = d.pop("id")

        linked_by = PushContactSubscriptionLinkedBy(d.pop("linkedBy"))




        platform = d.pop("platform")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        site = d.pop("site")

        time_zone = d.pop("timeZone")

        _last_seen = d.pop("lastSeen", UNSET)
        last_seen: datetime.datetime | Unset
        if isinstance(_last_seen,  Unset):
            last_seen = UNSET
        else:
            last_seen = datetime.datetime.fromisoformat(_last_seen)




        push_contact_subscription = cls(
            browser=browser,
            id=id,
            linked_by=linked_by,
            platform=platform,
            since=since,
            site=site,
            time_zone=time_zone,
            last_seen=last_seen,
        )

        return push_contact_subscription

