from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.placement_auth_count import PlacementAuthCount





T = TypeVar("T", bound="PlacementAuthSummary")



@_attrs_define
class PlacementAuthSummary:
    """ 
        Attributes:
            checked (int):
            dkim (PlacementAuthCount):
            dmarc (PlacementAuthCount):
            spf (PlacementAuthCount):
            dkim_domain (str | Unset):
            dkim_selector (str | Unset):
            spf_domain (str | Unset):
     """

    checked: int
    dkim: PlacementAuthCount
    dmarc: PlacementAuthCount
    spf: PlacementAuthCount
    dkim_domain: str | Unset = UNSET
    dkim_selector: str | Unset = UNSET
    spf_domain: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_auth_count import PlacementAuthCount # noqa: PLC0415
        checked = self.checked

        dkim = self.dkim.to_dict()

        dmarc = self.dmarc.to_dict()

        spf = self.spf.to_dict()

        dkim_domain = self.dkim_domain

        dkim_selector = self.dkim_selector

        spf_domain = self.spf_domain


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checked": checked,
            "dkim": dkim,
            "dmarc": dmarc,
            "spf": spf,
        })
        if dkim_domain is not UNSET:
            field_dict["dkimDomain"] = dkim_domain
        if dkim_selector is not UNSET:
            field_dict["dkimSelector"] = dkim_selector
        if spf_domain is not UNSET:
            field_dict["spfDomain"] = spf_domain

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_auth_count import PlacementAuthCount # noqa: PLC0415
        d = dict(src_dict)
        checked = d.pop("checked")

        dkim = PlacementAuthCount.from_dict(d.pop("dkim"))




        dmarc = PlacementAuthCount.from_dict(d.pop("dmarc"))




        spf = PlacementAuthCount.from_dict(d.pop("spf"))




        dkim_domain = d.pop("dkimDomain", UNSET)

        dkim_selector = d.pop("dkimSelector", UNSET)

        spf_domain = d.pop("spfDomain", UNSET)

        placement_auth_summary = cls(
            checked=checked,
            dkim=dkim,
            dmarc=dmarc,
            spf=spf,
            dkim_domain=dkim_domain,
            dkim_selector=dkim_selector,
            spf_domain=spf_domain,
        )

        return placement_auth_summary

