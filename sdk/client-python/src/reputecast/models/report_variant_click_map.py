from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.report_link import ReportLink
  from ..models.report_link_group import ReportLinkGroup
  from ..models.report_link_row import ReportLinkRow





T = TypeVar("T", bound="ReportVariantClickMap")



@_attrs_define
class ReportVariantClickMap:
    """ 
        Attributes:
            available (bool):
            bot_total (int):
            html (str):
            key (str):
            links (list[ReportLink]):
            rows (list[ReportLinkRow]):
            same_email (bool):
            total (int):
            other (ReportLinkGroup | Unset):
     """

    available: bool
    bot_total: int
    html: str
    key: str
    links: list[ReportLink]
    rows: list[ReportLinkRow]
    same_email: bool
    total: int
    other: ReportLinkGroup | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_link import ReportLink # noqa: PLC0415
        from ..models.report_link_group import ReportLinkGroup # noqa: PLC0415
        from ..models.report_link_row import ReportLinkRow # noqa: PLC0415
        available = self.available

        bot_total = self.bot_total

        html = self.html

        key = self.key

        links = []
        for links_item_data in self.links:
            links_item = links_item_data.to_dict()
            links.append(links_item)



        rows = []
        for rows_item_data in self.rows:
            rows_item = rows_item_data.to_dict()
            rows.append(rows_item)



        same_email = self.same_email

        total = self.total

        other: dict[str, Any] | Unset = UNSET
        if not isinstance(self.other, Unset):
            other = self.other.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "botTotal": bot_total,
            "html": html,
            "key": key,
            "links": links,
            "rows": rows,
            "sameEmail": same_email,
            "total": total,
        })
        if other is not UNSET:
            field_dict["other"] = other

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_link import ReportLink # noqa: PLC0415
        from ..models.report_link_group import ReportLinkGroup # noqa: PLC0415
        from ..models.report_link_row import ReportLinkRow # noqa: PLC0415
        d = dict(src_dict)
        available = d.pop("available")

        bot_total = d.pop("botTotal")

        html = d.pop("html")

        key = d.pop("key")

        links = []
        _links = d.pop("links")
        for links_item_data in (_links):
            links_item = ReportLink.from_dict(links_item_data)



            links.append(links_item)


        rows = []
        _rows = d.pop("rows")
        for rows_item_data in (_rows):
            rows_item = ReportLinkRow.from_dict(rows_item_data)



            rows.append(rows_item)


        same_email = d.pop("sameEmail")

        total = d.pop("total")

        _other = d.pop("other", UNSET)
        other: ReportLinkGroup | Unset
        if isinstance(_other,  Unset):
            other = UNSET
        else:
            other = ReportLinkGroup.from_dict(_other)




        report_variant_click_map = cls(
            available=available,
            bot_total=bot_total,
            html=html,
            key=key,
            links=links,
            rows=rows,
            same_email=same_email,
            total=total,
            other=other,
        )

        return report_variant_click_map

