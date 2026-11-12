from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_certificate_status import WebsiteCertificateStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WebsiteCertificate")



@_attrs_define
class WebsiteCertificate:
    """ 
        Attributes:
            status (WebsiteCertificateStatus):
            detail (str | Unset):
            issuer (str | Unset):
            renews_at (datetime.datetime | Unset):
     """

    status: WebsiteCertificateStatus
    detail: str | Unset = UNSET
    issuer: str | Unset = UNSET
    renews_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        status = self.status.value

        detail = self.detail

        issuer = self.issuer

        renews_at: str | Unset = UNSET
        if not isinstance(self.renews_at, Unset):
            renews_at = self.renews_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "status": status,
        })
        if detail is not UNSET:
            field_dict["detail"] = detail
        if issuer is not UNSET:
            field_dict["issuer"] = issuer
        if renews_at is not UNSET:
            field_dict["renewsAt"] = renews_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        status = WebsiteCertificateStatus(d.pop("status"))




        detail = d.pop("detail", UNSET)

        issuer = d.pop("issuer", UNSET)

        _renews_at = d.pop("renewsAt", UNSET)
        renews_at: datetime.datetime | Unset
        if isinstance(_renews_at,  Unset):
            renews_at = UNSET
        else:
            renews_at = datetime.datetime.fromisoformat(_renews_at)




        website_certificate = cls(
            status=status,
            detail=detail,
            issuer=issuer,
            renews_at=renews_at,
        )

        return website_certificate

