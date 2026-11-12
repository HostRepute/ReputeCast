from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.adapter_community_kind import AdapterCommunityKind
from ..models.adapter_community_update import AdapterCommunityUpdate
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AdapterCommunity")



@_attrs_define
class AdapterCommunity:
    """ 
        Attributes:
            capabilities (list[str]):
            connected (int):
            id (str):
            installed_at (datetime.datetime):
            kind (AdapterCommunityKind):
            maintainer (str):
            name (str):
            update (AdapterCommunityUpdate):
            version (str):
            last_release (datetime.datetime | Unset):
            latest (str | Unset):
            source_url (str | Unset):
     """

    capabilities: list[str]
    connected: int
    id: str
    installed_at: datetime.datetime
    kind: AdapterCommunityKind
    maintainer: str
    name: str
    update: AdapterCommunityUpdate
    version: str
    last_release: datetime.datetime | Unset = UNSET
    latest: str | Unset = UNSET
    source_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        capabilities = self.capabilities



        connected = self.connected

        id = self.id

        installed_at = self.installed_at.isoformat()

        kind = self.kind.value

        maintainer = self.maintainer

        name = self.name

        update = self.update.value

        version = self.version

        last_release: str | Unset = UNSET
        if not isinstance(self.last_release, Unset):
            last_release = self.last_release.isoformat()

        latest = self.latest

        source_url = self.source_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "capabilities": capabilities,
            "connected": connected,
            "id": id,
            "installedAt": installed_at,
            "kind": kind,
            "maintainer": maintainer,
            "name": name,
            "update": update,
            "version": version,
        })
        if last_release is not UNSET:
            field_dict["lastRelease"] = last_release
        if latest is not UNSET:
            field_dict["latest"] = latest
        if source_url is not UNSET:
            field_dict["sourceUrl"] = source_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        capabilities = cast(list[str], d.pop("capabilities"))


        connected = d.pop("connected")

        id = d.pop("id")

        installed_at = datetime.datetime.fromisoformat(d.pop("installedAt"))




        kind = AdapterCommunityKind(d.pop("kind"))




        maintainer = d.pop("maintainer")

        name = d.pop("name")

        update = AdapterCommunityUpdate(d.pop("update"))




        version = d.pop("version")

        _last_release = d.pop("lastRelease", UNSET)
        last_release: datetime.datetime | Unset
        if isinstance(_last_release,  Unset):
            last_release = UNSET
        else:
            last_release = datetime.datetime.fromisoformat(_last_release)




        latest = d.pop("latest", UNSET)

        source_url = d.pop("sourceUrl", UNSET)

        adapter_community = cls(
            capabilities=capabilities,
            connected=connected,
            id=id,
            installed_at=installed_at,
            kind=kind,
            maintainer=maintainer,
            name=name,
            update=update,
            version=version,
            last_release=last_release,
            latest=latest,
            source_url=source_url,
        )

        return adapter_community

