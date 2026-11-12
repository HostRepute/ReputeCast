from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DeliverabilityDomainAuth")



@_attrs_define
class DeliverabilityDomainAuth:
    """ 
        Attributes:
            bimi (str):
            dkim (str):
            dmarc (str): Policy (none, quarantine, reject) or missing / invalid
            return_path (str):
            spf (str):
            checked_at (datetime.datetime | Unset):
     """

    bimi: str
    dkim: str
    dmarc: str
    return_path: str
    spf: str
    checked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        bimi = self.bimi

        dkim = self.dkim

        dmarc = self.dmarc

        return_path = self.return_path

        spf = self.spf

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bimi": bimi,
            "dkim": dkim,
            "dmarc": dmarc,
            "returnPath": return_path,
            "spf": spf,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bimi = d.pop("bimi")

        dkim = d.pop("dkim")

        dmarc = d.pop("dmarc")

        return_path = d.pop("returnPath")

        spf = d.pop("spf")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        deliverability_domain_auth = cls(
            bimi=bimi,
            dkim=dkim,
            dmarc=dmarc,
            return_path=return_path,
            spf=spf,
            checked_at=checked_at,
        )

        return deliverability_domain_auth

