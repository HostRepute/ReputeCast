from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_address_domain_status import WebsiteAddressDomainStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_certificate import WebsiteCertificate





T = TypeVar("T", bound="WebsiteAddress")



@_attrs_define
class WebsiteAddress:
    """ 
        Attributes:
            certificate (WebsiteCertificate):
            domain (str):
            domain_detail (str):
            domain_status (WebsiteAddressDomainStatus):
            fallback (str):
            host (str):
            target (str):
            url (str):
            checked_at (datetime.datetime | Unset):
            verified_at (datetime.datetime | Unset):
     """

    certificate: WebsiteCertificate
    domain: str
    domain_detail: str
    domain_status: WebsiteAddressDomainStatus
    fallback: str
    host: str
    target: str
    url: str
    checked_at: datetime.datetime | Unset = UNSET
    verified_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_certificate import WebsiteCertificate # noqa: PLC0415
        certificate = self.certificate.to_dict()

        domain = self.domain

        domain_detail = self.domain_detail

        domain_status = self.domain_status.value

        fallback = self.fallback

        host = self.host

        target = self.target

        url = self.url

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        verified_at: str | Unset = UNSET
        if not isinstance(self.verified_at, Unset):
            verified_at = self.verified_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "certificate": certificate,
            "domain": domain,
            "domainDetail": domain_detail,
            "domainStatus": domain_status,
            "fallback": fallback,
            "host": host,
            "target": target,
            "url": url,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if verified_at is not UNSET:
            field_dict["verifiedAt"] = verified_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_certificate import WebsiteCertificate # noqa: PLC0415
        d = dict(src_dict)
        certificate = WebsiteCertificate.from_dict(d.pop("certificate"))




        domain = d.pop("domain")

        domain_detail = d.pop("domainDetail")

        domain_status = WebsiteAddressDomainStatus(d.pop("domainStatus"))




        fallback = d.pop("fallback")

        host = d.pop("host")

        target = d.pop("target")

        url = d.pop("url")

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




        website_address = cls(
            certificate=certificate,
            domain=domain,
            domain_detail=domain_detail,
            domain_status=domain_status,
            fallback=fallback,
            host=host,
            target=target,
            url=url,
            checked_at=checked_at,
            verified_at=verified_at,
        )

        return website_address

