from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.placement_counts import PlacementCounts





T = TypeVar("T", bound="PlacementMailboxResult")



@_attrs_define
class PlacementMailboxResult:
    """ 
        Attributes:
            counts (PlacementCounts):
            name (str): Gmail, Outlook.com, Yahoo
            provider (str): The seed mailbox provider's key (gmail)
            tab (str | Unset):
     """

    counts: PlacementCounts
    name: str
    provider: str
    tab: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_counts import PlacementCounts # noqa: PLC0415
        counts = self.counts.to_dict()

        name = self.name

        provider = self.provider

        tab = self.tab


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "counts": counts,
            "name": name,
            "provider": provider,
        })
        if tab is not UNSET:
            field_dict["tab"] = tab

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_counts import PlacementCounts # noqa: PLC0415
        d = dict(src_dict)
        counts = PlacementCounts.from_dict(d.pop("counts"))




        name = d.pop("name")

        provider = d.pop("provider")

        tab = d.pop("tab", UNSET)

        placement_mailbox_result = cls(
            counts=counts,
            name=name,
            provider=provider,
            tab=tab,
        )

        return placement_mailbox_result

