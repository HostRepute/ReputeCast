from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_check_key import PushCheckKey
from ..models.push_check_status import PushCheckStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.push_check_site import PushCheckSite
  from ..models.push_cut import PushCut
  from ..models.push_image import PushImage
  from ..models.push_zone import PushZone





T = TypeVar("T", bound="PushCheck")



@_attrs_define
class PushCheck:
    """ 
        Attributes:
            anonymous (int):
            at (datetime.datetime):
            at_zone (str):
            cap (int):
            count (int):
            cuts (list[PushCut]):
            had_today (int):
            held (int):
            held_zones (list[PushZone]):
            helpers (list[str]):
            inside (int):
            key (PushCheckKey):
            linked (int):
            message_length (int):
            missing (list[str]):
            other_sites (int):
            over_cap (int):
            quiet_off (bool):
            rules (list[str]):
            sites (list[PushCheckSite]):
            skipped (int):
            status (PushCheckStatus):
            title_length (int):
            window (str):
            author (str | Unset):
            author_role (str | Unset):
            image (PushImage | Unset):
            keys_accepted_at (datetime.datetime | Unset):
            keys_created_at (datetime.datetime | Unset):
            link (str | Unset):
            link_demo (bool | Unset):
            link_error (str | Unset):
            link_millis (int | Unset):
            link_status (int | Unset):
            ratio (str | Unset):
     """

    anonymous: int
    at: datetime.datetime
    at_zone: str
    cap: int
    count: int
    cuts: list[PushCut]
    had_today: int
    held: int
    held_zones: list[PushZone]
    helpers: list[str]
    inside: int
    key: PushCheckKey
    linked: int
    message_length: int
    missing: list[str]
    other_sites: int
    over_cap: int
    quiet_off: bool
    rules: list[str]
    sites: list[PushCheckSite]
    skipped: int
    status: PushCheckStatus
    title_length: int
    window: str
    author: str | Unset = UNSET
    author_role: str | Unset = UNSET
    image: PushImage | Unset = UNSET
    keys_accepted_at: datetime.datetime | Unset = UNSET
    keys_created_at: datetime.datetime | Unset = UNSET
    link: str | Unset = UNSET
    link_demo: bool | Unset = UNSET
    link_error: str | Unset = UNSET
    link_millis: int | Unset = UNSET
    link_status: int | Unset = UNSET
    ratio: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_check_site import PushCheckSite # noqa: PLC0415
        from ..models.push_cut import PushCut # noqa: PLC0415
        from ..models.push_image import PushImage # noqa: PLC0415
        from ..models.push_zone import PushZone # noqa: PLC0415
        anonymous = self.anonymous

        at = self.at.isoformat()

        at_zone = self.at_zone

        cap = self.cap

        count = self.count

        cuts = []
        for cuts_item_data in self.cuts:
            cuts_item = cuts_item_data.to_dict()
            cuts.append(cuts_item)



        had_today = self.had_today

        held = self.held

        held_zones = []
        for held_zones_item_data in self.held_zones:
            held_zones_item = held_zones_item_data.to_dict()
            held_zones.append(held_zones_item)



        helpers = self.helpers



        inside = self.inside

        key = self.key.value

        linked = self.linked

        message_length = self.message_length

        missing = self.missing



        other_sites = self.other_sites

        over_cap = self.over_cap

        quiet_off = self.quiet_off

        rules = self.rules



        sites = []
        for sites_item_data in self.sites:
            sites_item = sites_item_data.to_dict()
            sites.append(sites_item)



        skipped = self.skipped

        status = self.status.value

        title_length = self.title_length

        window = self.window

        author = self.author

        author_role = self.author_role

        image: dict[str, Any] | Unset = UNSET
        if not isinstance(self.image, Unset):
            image = self.image.to_dict()

        keys_accepted_at: str | Unset = UNSET
        if not isinstance(self.keys_accepted_at, Unset):
            keys_accepted_at = self.keys_accepted_at.isoformat()

        keys_created_at: str | Unset = UNSET
        if not isinstance(self.keys_created_at, Unset):
            keys_created_at = self.keys_created_at.isoformat()

        link = self.link

        link_demo = self.link_demo

        link_error = self.link_error

        link_millis = self.link_millis

        link_status = self.link_status

        ratio = self.ratio


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "anonymous": anonymous,
            "at": at,
            "atZone": at_zone,
            "cap": cap,
            "count": count,
            "cuts": cuts,
            "hadToday": had_today,
            "held": held,
            "heldZones": held_zones,
            "helpers": helpers,
            "inside": inside,
            "key": key,
            "linked": linked,
            "messageLength": message_length,
            "missing": missing,
            "otherSites": other_sites,
            "overCap": over_cap,
            "quietOff": quiet_off,
            "rules": rules,
            "sites": sites,
            "skipped": skipped,
            "status": status,
            "titleLength": title_length,
            "window": window,
        })
        if author is not UNSET:
            field_dict["author"] = author
        if author_role is not UNSET:
            field_dict["authorRole"] = author_role
        if image is not UNSET:
            field_dict["image"] = image
        if keys_accepted_at is not UNSET:
            field_dict["keysAcceptedAt"] = keys_accepted_at
        if keys_created_at is not UNSET:
            field_dict["keysCreatedAt"] = keys_created_at
        if link is not UNSET:
            field_dict["link"] = link
        if link_demo is not UNSET:
            field_dict["linkDemo"] = link_demo
        if link_error is not UNSET:
            field_dict["linkError"] = link_error
        if link_millis is not UNSET:
            field_dict["linkMillis"] = link_millis
        if link_status is not UNSET:
            field_dict["linkStatus"] = link_status
        if ratio is not UNSET:
            field_dict["ratio"] = ratio

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_check_site import PushCheckSite # noqa: PLC0415
        from ..models.push_cut import PushCut # noqa: PLC0415
        from ..models.push_image import PushImage # noqa: PLC0415
        from ..models.push_zone import PushZone # noqa: PLC0415
        d = dict(src_dict)
        anonymous = d.pop("anonymous")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        at_zone = d.pop("atZone")

        cap = d.pop("cap")

        count = d.pop("count")

        cuts = []
        _cuts = d.pop("cuts")
        for cuts_item_data in (_cuts):
            cuts_item = PushCut.from_dict(cuts_item_data)



            cuts.append(cuts_item)


        had_today = d.pop("hadToday")

        held = d.pop("held")

        held_zones = []
        _held_zones = d.pop("heldZones")
        for held_zones_item_data in (_held_zones):
            held_zones_item = PushZone.from_dict(held_zones_item_data)



            held_zones.append(held_zones_item)


        helpers = cast(list[str], d.pop("helpers"))


        inside = d.pop("inside")

        key = PushCheckKey(d.pop("key"))




        linked = d.pop("linked")

        message_length = d.pop("messageLength")

        missing = cast(list[str], d.pop("missing"))


        other_sites = d.pop("otherSites")

        over_cap = d.pop("overCap")

        quiet_off = d.pop("quietOff")

        rules = cast(list[str], d.pop("rules"))


        sites = []
        _sites = d.pop("sites")
        for sites_item_data in (_sites):
            sites_item = PushCheckSite.from_dict(sites_item_data)



            sites.append(sites_item)


        skipped = d.pop("skipped")

        status = PushCheckStatus(d.pop("status"))




        title_length = d.pop("titleLength")

        window = d.pop("window")

        author = d.pop("author", UNSET)

        author_role = d.pop("authorRole", UNSET)

        _image = d.pop("image", UNSET)
        image: PushImage | Unset
        if isinstance(_image,  Unset):
            image = UNSET
        else:
            image = PushImage.from_dict(_image)




        _keys_accepted_at = d.pop("keysAcceptedAt", UNSET)
        keys_accepted_at: datetime.datetime | Unset
        if isinstance(_keys_accepted_at,  Unset):
            keys_accepted_at = UNSET
        else:
            keys_accepted_at = datetime.datetime.fromisoformat(_keys_accepted_at)




        _keys_created_at = d.pop("keysCreatedAt", UNSET)
        keys_created_at: datetime.datetime | Unset
        if isinstance(_keys_created_at,  Unset):
            keys_created_at = UNSET
        else:
            keys_created_at = datetime.datetime.fromisoformat(_keys_created_at)




        link = d.pop("link", UNSET)

        link_demo = d.pop("linkDemo", UNSET)

        link_error = d.pop("linkError", UNSET)

        link_millis = d.pop("linkMillis", UNSET)

        link_status = d.pop("linkStatus", UNSET)

        ratio = d.pop("ratio", UNSET)

        push_check = cls(
            anonymous=anonymous,
            at=at,
            at_zone=at_zone,
            cap=cap,
            count=count,
            cuts=cuts,
            had_today=had_today,
            held=held,
            held_zones=held_zones,
            helpers=helpers,
            inside=inside,
            key=key,
            linked=linked,
            message_length=message_length,
            missing=missing,
            other_sites=other_sites,
            over_cap=over_cap,
            quiet_off=quiet_off,
            rules=rules,
            sites=sites,
            skipped=skipped,
            status=status,
            title_length=title_length,
            window=window,
            author=author,
            author_role=author_role,
            image=image,
            keys_accepted_at=keys_accepted_at,
            keys_created_at=keys_created_at,
            link=link,
            link_demo=link_demo,
            link_error=link_error,
            link_millis=link_millis,
            link_status=link_status,
            ratio=ratio,
        )

        return push_check

