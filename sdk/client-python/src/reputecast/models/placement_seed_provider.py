from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_seed_provider_oauth import PlacementSeedProviderOauth
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="PlacementSeedProvider")



@_attrs_define
class PlacementSeedProvider:
    """ 
        Attributes:
            company (str): Who runs the mailboxes (Apple for iCloud)
            host (str):
            inbox_name (str):
            key (str):
            name (str):
            password (str):
            port (int):
            spam_name (str):
            tabs (list[str]):
            username_label (str):
            imap_help (str | Unset):
            oauth (PlacementSeedProviderOauth | Unset):
            password_help (str | Unset):
     """

    company: str
    host: str
    inbox_name: str
    key: str
    name: str
    password: str
    port: int
    spam_name: str
    tabs: list[str]
    username_label: str
    imap_help: str | Unset = UNSET
    oauth: PlacementSeedProviderOauth | Unset = UNSET
    password_help: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        company = self.company

        host = self.host

        inbox_name = self.inbox_name

        key = self.key

        name = self.name

        password = self.password

        port = self.port

        spam_name = self.spam_name

        tabs = self.tabs



        username_label = self.username_label

        imap_help = self.imap_help

        oauth: str | Unset = UNSET
        if not isinstance(self.oauth, Unset):
            oauth = self.oauth.value


        password_help = self.password_help


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "company": company,
            "host": host,
            "inboxName": inbox_name,
            "key": key,
            "name": name,
            "password": password,
            "port": port,
            "spamName": spam_name,
            "tabs": tabs,
            "usernameLabel": username_label,
        })
        if imap_help is not UNSET:
            field_dict["imapHelp"] = imap_help
        if oauth is not UNSET:
            field_dict["oauth"] = oauth
        if password_help is not UNSET:
            field_dict["passwordHelp"] = password_help

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        company = d.pop("company")

        host = d.pop("host")

        inbox_name = d.pop("inboxName")

        key = d.pop("key")

        name = d.pop("name")

        password = d.pop("password")

        port = d.pop("port")

        spam_name = d.pop("spamName")

        tabs = cast(list[str], d.pop("tabs"))


        username_label = d.pop("usernameLabel")

        imap_help = d.pop("imapHelp", UNSET)

        _oauth = d.pop("oauth", UNSET)
        oauth: PlacementSeedProviderOauth | Unset
        if isinstance(_oauth,  Unset):
            oauth = UNSET
        else:
            oauth = PlacementSeedProviderOauth(_oauth)




        password_help = d.pop("passwordHelp", UNSET)

        placement_seed_provider = cls(
            company=company,
            host=host,
            inbox_name=inbox_name,
            key=key,
            name=name,
            password=password,
            port=port,
            spam_name=spam_name,
            tabs=tabs,
            username_label=username_label,
            imap_help=imap_help,
            oauth=oauth,
            password_help=password_help,
        )

        return placement_seed_provider

