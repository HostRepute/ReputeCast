from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.public_custom_domain_status import PublicCustomDomainStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PublicCustomDomain")



@_attrs_define
class PublicCustomDomain:
    """ 
        Attributes:
            detail (str):
            domain (str):  Example: news.acme.io.
            status (PublicCustomDomainStatus):
            target (str):  Example: reputecast.acme.io.
            txt_name (str):
            txt_value (str):
            checked_at (datetime.datetime | Unset):
            verified_at (datetime.datetime | Unset):
     """

    detail: str
    domain: str
    status: PublicCustomDomainStatus
    target: str
    txt_name: str
    txt_value: str
    checked_at: datetime.datetime | Unset = UNSET
    verified_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        detail = self.detail

        domain = self.domain

        status = self.status.value

        target = self.target

        txt_name = self.txt_name

        txt_value = self.txt_value

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        verified_at: str | Unset = UNSET
        if not isinstance(self.verified_at, Unset):
            verified_at = self.verified_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "detail": detail,
            "domain": domain,
            "status": status,
            "target": target,
            "txtName": txt_name,
            "txtValue": txt_value,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if verified_at is not UNSET:
            field_dict["verifiedAt"] = verified_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        detail = d.pop("detail")

        domain = d.pop("domain")

        status = PublicCustomDomainStatus(d.pop("status"))




        target = d.pop("target")

        txt_name = d.pop("txtName")

        txt_value = d.pop("txtValue")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        _verified_at = d.pop("verifiedAt", UNSET)
        verified_at: datetime.datetime | Unset
        if isinstance(_verified_at,  Unset):
            verified_at = UNSET
        else:
            verified_at = datetime.datetime.fromisoformat(_verified_at)




        public_custom_domain = cls(
            detail=detail,
            domain=domain,
            status=status,
            target=target,
            txt_name=txt_name,
            txt_value=txt_value,
            checked_at=checked_at,
            verified_at=verified_at,
        )

        return public_custom_domain

