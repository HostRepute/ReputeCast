from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_mailbox_input_after_processing import ConnectMailboxInputAfterProcessing
from ..models.connect_mailbox_input_poll_every import ConnectMailboxInputPollEvery
from ..models.connect_mailbox_input_protocol import ConnectMailboxInputProtocol
from ..models.connect_mailbox_input_tls import ConnectMailboxInputTls
from ..types import UNSET, Unset






T = TypeVar("T", bound="ConnectMailboxInput")



@_attrs_define
class ConnectMailboxInput:
    """ 
        Attributes:
            address (str): The return-path address bounces go to
            host (str):
            port (int):
            protocol (ConnectMailboxInputProtocol):
            tls (ConnectMailboxInputTls):
            username (str):
            after_processing (ConnectMailboxInputAfterProcessing | Unset):
            folder (str | Unset):
            password (str | Unset):
            poll_every (ConnectMailboxInputPollEvery | Unset):
     """

    address: str
    host: str
    port: int
    protocol: ConnectMailboxInputProtocol
    tls: ConnectMailboxInputTls
    username: str
    after_processing: ConnectMailboxInputAfterProcessing | Unset = UNSET
    folder: str | Unset = UNSET
    password: str | Unset = UNSET
    poll_every: ConnectMailboxInputPollEvery | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        address = self.address

        host = self.host

        port = self.port

        protocol = self.protocol.value

        tls = self.tls.value

        username = self.username

        after_processing: str | Unset = UNSET
        if not isinstance(self.after_processing, Unset):
            after_processing = self.after_processing.value


        folder = self.folder

        password = self.password

        poll_every: str | Unset = UNSET
        if not isinstance(self.poll_every, Unset):
            poll_every = self.poll_every.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "address": address,
            "host": host,
            "port": port,
            "protocol": protocol,
            "tls": tls,
            "username": username,
        })
        if after_processing is not UNSET:
            field_dict["afterProcessing"] = after_processing
        if folder is not UNSET:
            field_dict["folder"] = folder
        if password is not UNSET:
            field_dict["password"] = password
        if poll_every is not UNSET:
            field_dict["pollEvery"] = poll_every

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        address = d.pop("address")

        host = d.pop("host")

        port = d.pop("port")

        protocol = ConnectMailboxInputProtocol(d.pop("protocol"))




        tls = ConnectMailboxInputTls(d.pop("tls"))




        username = d.pop("username")

        _after_processing = d.pop("afterProcessing", UNSET)
        after_processing: ConnectMailboxInputAfterProcessing | Unset
        if isinstance(_after_processing,  Unset):
            after_processing = UNSET
        else:
            after_processing = ConnectMailboxInputAfterProcessing(_after_processing)




        folder = d.pop("folder", UNSET)

        password = d.pop("password", UNSET)

        _poll_every = d.pop("pollEvery", UNSET)
        poll_every: ConnectMailboxInputPollEvery | Unset
        if isinstance(_poll_every,  Unset):
            poll_every = UNSET
        else:
            poll_every = ConnectMailboxInputPollEvery(_poll_every)




        connect_mailbox_input = cls(
            address=address,
            host=host,
            port=port,
            protocol=protocol,
            tls=tls,
            username=username,
            after_processing=after_processing,
            folder=folder,
            password=password,
            poll_every=poll_every,
        )

        return connect_mailbox_input

