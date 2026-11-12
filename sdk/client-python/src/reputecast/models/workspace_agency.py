from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.workspace_agency_kind import WorkspaceAgencyKind
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.agency_person_ref import AgencyPersonRef
  from ..models.agency_rate import AgencyRate
  from ..models.workspace_agency_staff import WorkspaceAgencyStaff





T = TypeVar("T", bound="WorkspaceAgency")



@_attrs_define
class WorkspaceAgency:
    """ 
        Attributes:
            billed (bool):
            can_save_template (bool):
            managed (bool):
            staff (list[WorkspaceAgencyStaff]):
            viewer_is_staff (bool):
            agency (str | Unset):
            agency_initials (str | Unset):
            client_since (datetime.datetime | Unset):
            contact (AgencyPersonRef | Unset):
            host (str | Unset):
            kind (WorkspaceAgencyKind | Unset):
            rate (AgencyRate | Unset):
            rate_since (datetime.datetime | Unset):
            template (str | Unset):
     """

    billed: bool
    can_save_template: bool
    managed: bool
    staff: list[WorkspaceAgencyStaff]
    viewer_is_staff: bool
    agency: str | Unset = UNSET
    agency_initials: str | Unset = UNSET
    client_since: datetime.datetime | Unset = UNSET
    contact: AgencyPersonRef | Unset = UNSET
    host: str | Unset = UNSET
    kind: WorkspaceAgencyKind | Unset = UNSET
    rate: AgencyRate | Unset = UNSET
    rate_since: datetime.datetime | Unset = UNSET
    template: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.agency_person_ref import AgencyPersonRef # noqa: PLC0415
        from ..models.agency_rate import AgencyRate # noqa: PLC0415
        from ..models.workspace_agency_staff import WorkspaceAgencyStaff # noqa: PLC0415
        billed = self.billed

        can_save_template = self.can_save_template

        managed = self.managed

        staff = []
        for staff_item_data in self.staff:
            staff_item = staff_item_data.to_dict()
            staff.append(staff_item)



        viewer_is_staff = self.viewer_is_staff

        agency = self.agency

        agency_initials = self.agency_initials

        client_since: str | Unset = UNSET
        if not isinstance(self.client_since, Unset):
            client_since = self.client_since.isoformat()

        contact: dict[str, Any] | Unset = UNSET
        if not isinstance(self.contact, Unset):
            contact = self.contact.to_dict()

        host = self.host

        kind: str | Unset = UNSET
        if not isinstance(self.kind, Unset):
            kind = self.kind.value


        rate: dict[str, Any] | Unset = UNSET
        if not isinstance(self.rate, Unset):
            rate = self.rate.to_dict()

        rate_since: str | Unset = UNSET
        if not isinstance(self.rate_since, Unset):
            rate_since = self.rate_since.isoformat()

        template = self.template


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "billed": billed,
            "canSaveTemplate": can_save_template,
            "managed": managed,
            "staff": staff,
            "viewerIsStaff": viewer_is_staff,
        })
        if agency is not UNSET:
            field_dict["agency"] = agency
        if agency_initials is not UNSET:
            field_dict["agencyInitials"] = agency_initials
        if client_since is not UNSET:
            field_dict["clientSince"] = client_since
        if contact is not UNSET:
            field_dict["contact"] = contact
        if host is not UNSET:
            field_dict["host"] = host
        if kind is not UNSET:
            field_dict["kind"] = kind
        if rate is not UNSET:
            field_dict["rate"] = rate
        if rate_since is not UNSET:
            field_dict["rateSince"] = rate_since
        if template is not UNSET:
            field_dict["template"] = template

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.agency_person_ref import AgencyPersonRef # noqa: PLC0415
        from ..models.agency_rate import AgencyRate # noqa: PLC0415
        from ..models.workspace_agency_staff import WorkspaceAgencyStaff # noqa: PLC0415
        d = dict(src_dict)
        billed = d.pop("billed")

        can_save_template = d.pop("canSaveTemplate")

        managed = d.pop("managed")

        staff = []
        _staff = d.pop("staff")
        for staff_item_data in (_staff):
            staff_item = WorkspaceAgencyStaff.from_dict(staff_item_data)



            staff.append(staff_item)


        viewer_is_staff = d.pop("viewerIsStaff")

        agency = d.pop("agency", UNSET)

        agency_initials = d.pop("agencyInitials", UNSET)

        _client_since = d.pop("clientSince", UNSET)
        client_since: datetime.datetime | Unset
        if isinstance(_client_since,  Unset):
            client_since = UNSET
        else:
            client_since = datetime.datetime.fromisoformat(_client_since)




        _contact = d.pop("contact", UNSET)
        contact: AgencyPersonRef | Unset
        if isinstance(_contact,  Unset):
            contact = UNSET
        else:
            contact = AgencyPersonRef.from_dict(_contact)




        host = d.pop("host", UNSET)

        _kind = d.pop("kind", UNSET)
        kind: WorkspaceAgencyKind | Unset
        if isinstance(_kind,  Unset):
            kind = UNSET
        else:
            kind = WorkspaceAgencyKind(_kind)




        _rate = d.pop("rate", UNSET)
        rate: AgencyRate | Unset
        if isinstance(_rate,  Unset):
            rate = UNSET
        else:
            rate = AgencyRate.from_dict(_rate)




        _rate_since = d.pop("rateSince", UNSET)
        rate_since: datetime.datetime | Unset
        if isinstance(_rate_since,  Unset):
            rate_since = UNSET
        else:
            rate_since = datetime.datetime.fromisoformat(_rate_since)




        template = d.pop("template", UNSET)

        workspace_agency = cls(
            billed=billed,
            can_save_template=can_save_template,
            managed=managed,
            staff=staff,
            viewer_is_staff=viewer_is_staff,
            agency=agency,
            agency_initials=agency_initials,
            client_since=client_since,
            contact=contact,
            host=host,
            kind=kind,
            rate=rate,
            rate_since=rate_since,
            template=template,
        )

        return workspace_agency

