from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_short_domain_status import SmsShortDomainStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsShortDomain")



@_attrs_define
class SmsShortDomain:
    """ 
        Attributes:
            domain (str):
            status (SmsShortDomainStatus):
            target (str):
            txt_name (str):
            txt_value (str):
            checked_at (datetime.datetime | Unset):
            detail (str | Unset):
            verified_at (datetime.datetime | Unset):
     """

    domain: str
    status: SmsShortDomainStatus
    target: str
    txt_name: str
    txt_value: str
    checked_at: datetime.datetime | Unset = UNSET
    detail: str | Unset = UNSET
    verified_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        domain = self.domain

        status = self.status.value

        target = self.target

        txt_name = self.txt_name

        txt_value = self.txt_value

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        detail = self.detail

        verified_at: str | Unset = UNSET
        if not isinstance(self.verified_at, Unset):
            verified_at = self.verified_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "status": status,
            "target": target,
            "txtName": txt_name,
            "txtValue": txt_value,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if detail is not UNSET:
            field_dict["detail"] = detail
        if verified_at is not UNSET:
            field_dict["verifiedAt"] = verified_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        domain = d.pop("domain")

        status = SmsShortDomainStatus(d.pop("status"))




        target = d.pop("target")

        txt_name = d.pop("txtName")

        txt_value = d.pop("txtValue")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        detail = d.pop("detail", UNSET)

        _verified_at = d.pop("verifiedAt", UNSET)
        verified_at: datetime.datetime | Unset
        if isinstance(_verified_at,  Unset):
            verified_at = UNSET
        else:
            verified_at = datetime.datetime.fromisoformat(_verified_at)




        sms_short_domain = cls(
            domain=domain,
            status=status,
            target=target,
            txt_name=txt_name,
            txt_value=txt_value,
            checked_at=checked_at,
            detail=detail,
            verified_at=verified_at,
        )

        return sms_short_domain

