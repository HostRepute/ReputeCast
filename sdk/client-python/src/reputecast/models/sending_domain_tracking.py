from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sending_domain_tracking_status import SendingDomainTrackingStatus






T = TypeVar("T", bound="SendingDomainTracking")



@_attrs_define
class SendingDomainTracking:
    """ 
        Attributes:
            detail (str):
            domain (str):
            status (SendingDomainTrackingStatus):
            suggested (str):
            target (str):
            txt_name (str):
            txt_value (str):
     """

    detail: str
    domain: str
    status: SendingDomainTrackingStatus
    suggested: str
    target: str
    txt_name: str
    txt_value: str





    def to_dict(self) -> dict[str, Any]:
        detail = self.detail

        domain = self.domain

        status = self.status.value

        suggested = self.suggested

        target = self.target

        txt_name = self.txt_name

        txt_value = self.txt_value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "detail": detail,
            "domain": domain,
            "status": status,
            "suggested": suggested,
            "target": target,
            "txtName": txt_name,
            "txtValue": txt_value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        detail = d.pop("detail")

        domain = d.pop("domain")

        status = SendingDomainTrackingStatus(d.pop("status"))




        suggested = d.pop("suggested")

        target = d.pop("target")

        txt_name = d.pop("txtName")

        txt_value = d.pop("txtValue")

        sending_domain_tracking = cls(
            detail=detail,
            domain=domain,
            status=status,
            suggested=suggested,
            target=target,
            txt_name=txt_name,
            txt_value=txt_value,
        )

        return sending_domain_tracking

