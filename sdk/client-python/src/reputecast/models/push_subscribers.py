from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.push_count import PushCount
  from ..models.push_list_highlight import PushListHighlight





T = TypeVar("T", bound="PushSubscribers")



@_attrs_define
class PushSubscribers:
    """ 
        Attributes:
            anonymous (int):
            by_browser (list[PushCount]):
            by_site (list[PushCount]):
            linked (int):
            total (int):
            highlight (PushListHighlight | Unset):
     """

    anonymous: int
    by_browser: list[PushCount]
    by_site: list[PushCount]
    linked: int
    total: int
    highlight: PushListHighlight | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_count import PushCount # noqa: PLC0415
        from ..models.push_list_highlight import PushListHighlight # noqa: PLC0415
        anonymous = self.anonymous

        by_browser = []
        for by_browser_item_data in self.by_browser:
            by_browser_item = by_browser_item_data.to_dict()
            by_browser.append(by_browser_item)



        by_site = []
        for by_site_item_data in self.by_site:
            by_site_item = by_site_item_data.to_dict()
            by_site.append(by_site_item)



        linked = self.linked

        total = self.total

        highlight: dict[str, Any] | Unset = UNSET
        if not isinstance(self.highlight, Unset):
            highlight = self.highlight.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "anonymous": anonymous,
            "byBrowser": by_browser,
            "bySite": by_site,
            "linked": linked,
            "total": total,
        })
        if highlight is not UNSET:
            field_dict["highlight"] = highlight

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_count import PushCount # noqa: PLC0415
        from ..models.push_list_highlight import PushListHighlight # noqa: PLC0415
        d = dict(src_dict)
        anonymous = d.pop("anonymous")

        by_browser = []
        _by_browser = d.pop("byBrowser")
        for by_browser_item_data in (_by_browser):
            by_browser_item = PushCount.from_dict(by_browser_item_data)



            by_browser.append(by_browser_item)


        by_site = []
        _by_site = d.pop("bySite")
        for by_site_item_data in (_by_site):
            by_site_item = PushCount.from_dict(by_site_item_data)



            by_site.append(by_site_item)


        linked = d.pop("linked")

        total = d.pop("total")

        _highlight = d.pop("highlight", UNSET)
        highlight: PushListHighlight | Unset
        if isinstance(_highlight,  Unset):
            highlight = UNSET
        else:
            highlight = PushListHighlight.from_dict(_highlight)




        push_subscribers = cls(
            anonymous=anonymous,
            by_browser=by_browser,
            by_site=by_site,
            linked=linked,
            total=total,
            highlight=highlight,
        )

        return push_subscribers

