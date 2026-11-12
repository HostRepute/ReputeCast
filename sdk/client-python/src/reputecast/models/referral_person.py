from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.referral_person_link_status import ReferralPersonLinkStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReferralPerson")



@_attrs_define
class ReferralPerson:
    """ 
        Attributes:
            contact_id (str):
            counted (int):
            email (str):
            first_name (str):
            link_id (str):
            link_status (ReferralPersonLinkStatus):
            name (str):
            since (datetime.datetime | Unset): On the program's list since
     """

    contact_id: str
    counted: int
    email: str
    first_name: str
    link_id: str
    link_status: ReferralPersonLinkStatus
    name: str
    since: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        contact_id = self.contact_id

        counted = self.counted

        email = self.email

        first_name = self.first_name

        link_id = self.link_id

        link_status = self.link_status.value

        name = self.name

        since: str | Unset = UNSET
        if not isinstance(self.since, Unset):
            since = self.since.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contactId": contact_id,
            "counted": counted,
            "email": email,
            "firstName": first_name,
            "linkId": link_id,
            "linkStatus": link_status,
            "name": name,
        })
        if since is not UNSET:
            field_dict["since"] = since

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contact_id = d.pop("contactId")

        counted = d.pop("counted")

        email = d.pop("email")

        first_name = d.pop("firstName")

        link_id = d.pop("linkId")

        link_status = ReferralPersonLinkStatus(d.pop("linkStatus"))




        name = d.pop("name")

        _since = d.pop("since", UNSET)
        since: datetime.datetime | Unset
        if isinstance(_since,  Unset):
            since = UNSET
        else:
            since = datetime.datetime.fromisoformat(_since)




        referral_person = cls(
            contact_id=contact_id,
            counted=counted,
            email=email,
            first_name=first_name,
            link_id=link_id,
            link_status=link_status,
            name=name,
            since=since,
        )

        return referral_person

