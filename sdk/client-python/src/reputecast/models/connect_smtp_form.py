from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_smtp_form_after_processing import ConnectSMTPFormAfterProcessing
from ..models.connect_smtp_form_mailbox_tls import ConnectSMTPFormMailboxTls
from ..models.connect_smtp_form_poll_every import ConnectSMTPFormPollEvery
from ..models.connect_smtp_form_protocol import ConnectSMTPFormProtocol
from ..models.connect_smtp_form_tls import ConnectSMTPFormTls






T = TypeVar("T", bound="ConnectSMTPForm")



@_attrs_define
class ConnectSMTPForm:
    """ 
        Attributes:
            after_processing (ConnectSMTPFormAfterProcessing):
            daily_cap (str):
            folder (str):
            host (str):
            mailbox_host (str):
            mailbox_password_set (bool):
            mailbox_port (int):
            mailbox_tls (ConnectSMTPFormMailboxTls):
            mailbox_user (str):
            max_rate (str):
            password_set (bool):
            poll_every (ConnectSMTPFormPollEvery):
            port (int):
            protocol (ConnectSMTPFormProtocol):
            return_path (str):
            tls (ConnectSMTPFormTls):
            username (str):
     """

    after_processing: ConnectSMTPFormAfterProcessing
    daily_cap: str
    folder: str
    host: str
    mailbox_host: str
    mailbox_password_set: bool
    mailbox_port: int
    mailbox_tls: ConnectSMTPFormMailboxTls
    mailbox_user: str
    max_rate: str
    password_set: bool
    poll_every: ConnectSMTPFormPollEvery
    port: int
    protocol: ConnectSMTPFormProtocol
    return_path: str
    tls: ConnectSMTPFormTls
    username: str





    def to_dict(self) -> dict[str, Any]:
        after_processing = self.after_processing.value

        daily_cap = self.daily_cap

        folder = self.folder

        host = self.host

        mailbox_host = self.mailbox_host

        mailbox_password_set = self.mailbox_password_set

        mailbox_port = self.mailbox_port

        mailbox_tls = self.mailbox_tls.value

        mailbox_user = self.mailbox_user

        max_rate = self.max_rate

        password_set = self.password_set

        poll_every = self.poll_every.value

        port = self.port

        protocol = self.protocol.value

        return_path = self.return_path

        tls = self.tls.value

        username = self.username


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "afterProcessing": after_processing,
            "dailyCap": daily_cap,
            "folder": folder,
            "host": host,
            "mailboxHost": mailbox_host,
            "mailboxPasswordSet": mailbox_password_set,
            "mailboxPort": mailbox_port,
            "mailboxTls": mailbox_tls,
            "mailboxUser": mailbox_user,
            "maxRate": max_rate,
            "passwordSet": password_set,
            "pollEvery": poll_every,
            "port": port,
            "protocol": protocol,
            "returnPath": return_path,
            "tls": tls,
            "username": username,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        after_processing = ConnectSMTPFormAfterProcessing(d.pop("afterProcessing"))




        daily_cap = d.pop("dailyCap")

        folder = d.pop("folder")

        host = d.pop("host")

        mailbox_host = d.pop("mailboxHost")

        mailbox_password_set = d.pop("mailboxPasswordSet")

        mailbox_port = d.pop("mailboxPort")

        mailbox_tls = ConnectSMTPFormMailboxTls(d.pop("mailboxTls"))




        mailbox_user = d.pop("mailboxUser")

        max_rate = d.pop("maxRate")

        password_set = d.pop("passwordSet")

        poll_every = ConnectSMTPFormPollEvery(d.pop("pollEvery"))




        port = d.pop("port")

        protocol = ConnectSMTPFormProtocol(d.pop("protocol"))




        return_path = d.pop("returnPath")

        tls = ConnectSMTPFormTls(d.pop("tls"))




        username = d.pop("username")

        connect_smtp_form = cls(
            after_processing=after_processing,
            daily_cap=daily_cap,
            folder=folder,
            host=host,
            mailbox_host=mailbox_host,
            mailbox_password_set=mailbox_password_set,
            mailbox_port=mailbox_port,
            mailbox_tls=mailbox_tls,
            mailbox_user=mailbox_user,
            max_rate=max_rate,
            password_set=password_set,
            poll_every=poll_every,
            port=port,
            protocol=protocol,
            return_path=return_path,
            tls=tls,
            username=username,
        )

        return connect_smtp_form

