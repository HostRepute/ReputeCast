from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_seed_view_auth import PlacementSeedViewAuth
from ..models.placement_seed_view_provider import PlacementSeedViewProvider
from ..models.placement_seed_view_status import PlacementSeedViewStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.placement_folder import PlacementFolder





T = TypeVar("T", bound="PlacementSeedView")



@_attrs_define
class PlacementSeedView:
    """ 
        Attributes:
            added_at (datetime.datetime):
            added_by (str):
            auth (PlacementSeedViewAuth):
            company (str):
            email (str):
            folders (list[PlacementFolder]):
            id (str):
            password_label (str):
            provider (PlacementSeedViewProvider):
            provider_name (str):
            status (PlacementSeedViewStatus):
            checked_at (datetime.datetime | Unset):
            host (str | Unset):
            last_test (str | Unset):
            last_test_id (str | Unset):
            last_used_at (datetime.datetime | Unset):
            port (int | Unset):
            status_at (datetime.datetime | Unset):
            username (str | Unset):
     """

    added_at: datetime.datetime
    added_by: str
    auth: PlacementSeedViewAuth
    company: str
    email: str
    folders: list[PlacementFolder]
    id: str
    password_label: str
    provider: PlacementSeedViewProvider
    provider_name: str
    status: PlacementSeedViewStatus
    checked_at: datetime.datetime | Unset = UNSET
    host: str | Unset = UNSET
    last_test: str | Unset = UNSET
    last_test_id: str | Unset = UNSET
    last_used_at: datetime.datetime | Unset = UNSET
    port: int | Unset = UNSET
    status_at: datetime.datetime | Unset = UNSET
    username: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_folder import PlacementFolder # noqa: PLC0415
        added_at = self.added_at.isoformat()

        added_by = self.added_by

        auth = self.auth.value

        company = self.company

        email = self.email

        folders = []
        for folders_item_data in self.folders:
            folders_item = folders_item_data.to_dict()
            folders.append(folders_item)



        id = self.id

        password_label = self.password_label

        provider = self.provider.value

        provider_name = self.provider_name

        status = self.status.value

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        host = self.host

        last_test = self.last_test

        last_test_id = self.last_test_id

        last_used_at: str | Unset = UNSET
        if not isinstance(self.last_used_at, Unset):
            last_used_at = self.last_used_at.isoformat()

        port = self.port

        status_at: str | Unset = UNSET
        if not isinstance(self.status_at, Unset):
            status_at = self.status_at.isoformat()

        username = self.username


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "addedAt": added_at,
            "addedBy": added_by,
            "auth": auth,
            "company": company,
            "email": email,
            "folders": folders,
            "id": id,
            "passwordLabel": password_label,
            "provider": provider,
            "providerName": provider_name,
            "status": status,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if host is not UNSET:
            field_dict["host"] = host
        if last_test is not UNSET:
            field_dict["lastTest"] = last_test
        if last_test_id is not UNSET:
            field_dict["lastTestId"] = last_test_id
        if last_used_at is not UNSET:
            field_dict["lastUsedAt"] = last_used_at
        if port is not UNSET:
            field_dict["port"] = port
        if status_at is not UNSET:
            field_dict["statusAt"] = status_at
        if username is not UNSET:
            field_dict["username"] = username

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_folder import PlacementFolder # noqa: PLC0415
        d = dict(src_dict)
        added_at = datetime.datetime.fromisoformat(d.pop("addedAt"))




        added_by = d.pop("addedBy")

        auth = PlacementSeedViewAuth(d.pop("auth"))




        company = d.pop("company")

        email = d.pop("email")

        folders = []
        _folders = d.pop("folders")
        for folders_item_data in (_folders):
            folders_item = PlacementFolder.from_dict(folders_item_data)



            folders.append(folders_item)


        id = d.pop("id")

        password_label = d.pop("passwordLabel")

        provider = PlacementSeedViewProvider(d.pop("provider"))




        provider_name = d.pop("providerName")

        status = PlacementSeedViewStatus(d.pop("status"))




        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        host = d.pop("host", UNSET)

        last_test = d.pop("lastTest", UNSET)

        last_test_id = d.pop("lastTestId", UNSET)

        _last_used_at = d.pop("lastUsedAt", UNSET)
        last_used_at: datetime.datetime | Unset
        if isinstance(_last_used_at,  Unset):
            last_used_at = UNSET
        else:
            last_used_at = datetime.datetime.fromisoformat(_last_used_at)




        port = d.pop("port", UNSET)

        _status_at = d.pop("statusAt", UNSET)
        status_at: datetime.datetime | Unset
        if isinstance(_status_at,  Unset):
            status_at = UNSET
        else:
            status_at = datetime.datetime.fromisoformat(_status_at)




        username = d.pop("username", UNSET)

        placement_seed_view = cls(
            added_at=added_at,
            added_by=added_by,
            auth=auth,
            company=company,
            email=email,
            folders=folders,
            id=id,
            password_label=password_label,
            provider=provider,
            provider_name=provider_name,
            status=status,
            checked_at=checked_at,
            host=host,
            last_test=last_test,
            last_test_id=last_test_id,
            last_used_at=last_used_at,
            port=port,
            status_at=status_at,
            username=username,
        )

        return placement_seed_view

