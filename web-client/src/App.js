import React, { Component } from 'react';
import './App.css';
import { Avatar, Button, Layout, Menu, Icon, Input, Row, Col } from 'antd';

const {
  Header, Content, Footer, Sider,
} = Layout;

const { TextArea } = Input;

function colorForString(str) {
  var colorList = [
      '#332626',
      '#cc8533',
      '#a68500',
      '#245900',
      '#4d998a',
      '#002966',
      '#140033',
      '#331a2e',
      '#cc0036',
      '#993626',
      '#33210d',
      '#becc00',
      '#2ca600',
      '#103d40',
      '#265499',
      '#8f66cc',
      '#59003c',
      '#401016',
      '#734939',
      '#b39e86',
      '#a1b359',
      '#1d331a',
      '#33adcc',
      '#8f9cbf',
      '#4f4359',
      '#cc3399',
      '#b25965',
      '#bf4d00',
      '#734d00',
      '#657356',
      '#66cc8f',
      '#23698c',
      '#3600cc',
      '#550080',
    '#a67c8d' ];

    function hashString(s) {
      var hash = 0;
      var i;
      var chr;
      var len;
      if (!s || s.length === 0) {
        return hash;
      }
      for (i = 0, len = s.length; i < len; i++) {
        chr = s.charCodeAt(i);
        /* jshint ignore:start */ // JShint doesn't like abusing bitwise operaters
        hash = ((hash << 5) - hash) + chr;
        hash |= 0; // Convert to 32bit integer
        /* jshint ignore:end */
      }
      return Math.abs(hash);
    }

    return colorList[hashString(str) % colorList.length];
  };

class App extends Component {
  state = {
    user: process.env.REACT_APP_USERNAME,
    message: '',
    selectedRoom: 'General',
    rooms: ['General'],
    posts: [
      /*
      {User: 'user 1', Message: 'my message', PostedAt: (new Date().toJSON())},
      {User: 'user 2', Message: 'my message', PostedAt: (new Date().toJSON())},
      {User: 'user 1', Message: 'my message', PostedAt: (new Date().toJSON())},
      {User: 'user 3', Message: 'my message', PostedAt: (new Date().toJSON())},
      {User: 'user 4', Message: 'my message', PostedAt: (new Date().toJSON())},
      {User: 'user 40', Message: 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.', PostedAt: (new Date().toJSON())}, */
    ],
  }

  componentDidMount() {
    this.selectedRoomChanged();
    fetch(`http://127.0.0.1:${process.env.REACT_APP_SERVER_PORT}/rooms`)
      .then((response) => {
        return response.json();
      }).then((rooms) => {
        this.setState({
          rooms: Array.from(new Set(this.state.rooms.concat(rooms))),
        });
      });

    const socket = new WebSocket(`ws://127.0.0.1:${process.env.REACT_APP_SERVER_PORT}/ws`);

    // Connection opened
    socket.addEventListener('open', function (event) {
      console.log('WS Open');
    });

    // Listen for messages
    socket.addEventListener('message', (event) => {
      console.log('Message from server ', event.data);
      const msg = JSON.parse(event.data);
      this.setState({
        rooms: Array.from(new Set(this.state.rooms.concat(msg.RoomName))),
      });
      if (msg && msg.RoomName === this.state.selectedRoom) {
        this.setState((prev) => {
          if (this.state.selectedRoom !== msg.RoomName) {
            return prev;
          }
          return {
            posts: prev.posts.concat(msg.Post),
          };
        });
      }
    });
  }

  handleUserChanged = (e) => {
    this.setState({
      user: e.target.value,
    });
  }

  handleInputChanged = (e) => {
    this.setState({
      message: e.target.value,
    });
  }

  handlePostClicked = (e) => {
    fetch(`http://127.0.0.1:${process.env.REACT_APP_SERVER_PORT}/rooms/${this.state.selectedRoom}`, {
      method: "POST",
      mode: "cors",
      body: JSON.stringify({User: this.state.user, Message: this.state.message}),
    }).catch((e) => {
      console.error(e);
    });
  }

  selectedRoomChanged = () => {
    const room = this.state.selectedRoom;
    fetch(`http://127.0.0.1:${process.env.REACT_APP_SERVER_PORT}/rooms/${encodeURIComponent(room)}`)
      .then((response) => {
        return response.json();
      }).then((posts) => {
        this.setState({
          posts,
        });
      });
  }

  handleMenuClicked = (item) => {
    if (item.key === 'new-room-button') {
      const roomName = prompt("Name of new room?");
      this.setState({
        rooms: Array.from(new Set(this.state.rooms.concat(roomName))),
        selectedRoom: roomName,
      }, () => {
        this.selectedRoomChanged();
      });
      return;
    }
    this.setState({selectedRoom: item.key, posts: []}, () => {
      this.selectedRoomChanged();
    });
  }

  render() {
    return (
      <Layout>
        <Sider style={{
          overflow: 'auto', height: '100vh', position: 'fixed', left: 0,
        }}
        >
          <div className="logo" />
          <Menu theme="dark" mode="inline" selectedKeys={[this.state.selectedRoom]} onClick={this.handleMenuClicked}>
            {this.state.rooms.map((room, idx) => {
              return (
              <Menu.Item key={room}>
                <Icon type="team" />
                <span className="nav-text">{room}</span>
                </Menu.Item>
              );
            })}
            <Menu.Item key="new-room-button">
              <Button type="primary" ghost>New Chat Room</Button>
            </Menu.Item>
          </Menu>
        </Sider>
        <Layout style={{ marginLeft: 200 }}>
          <Header style={{ fontSize: '20px', background: '#fff', padding: '0 0 0 20px' }}>
            {this.state.selectedRoom}
          </Header>
          {this.state.posts.map((post, idx) => {
            return (
              <Content key={idx} style={{ margin: '15px 16px 0', overflow: 'initial' }}>
                <div style={{ padding: 5, background: '#fff' , minHeight: '35px' }}>
                  <Avatar style={{ backgroundColor: colorForString(post.User) }} icon="user" />
                  <strong style={{ color: colorForString(post.User), fontSize: 20, fontWeight: 900, paddingLeft: 10 }}>{post.User}</strong>
                  <small style={{ float: 'right' }}>{post.PostedAt}</small>
                  <br/>
                  {post.Message}
                </div>
              </Content>
            );
          })}
          <Footer>
            <Row>
              <Col span={12}>
                <TextArea placeholder="Enter a new message here..." rows={2} onChange={this.handleInputChanged} />
              </Col>
              <Col span={1} />
              <Col span={4}>
                <Button type="primary" onClick={this.handlePostClicked}>Post!</Button>
              </Col>
            </Row>
          </Footer>
          <Footer>
            Your username:
            <Input defaultValue={this.state.user} onChange={this.handleUserChanged} />
            <br/>
            Connected to: http://127.0.0.1:{process.env.REACT_APP_SERVER_PORT}
          </Footer>
        </Layout>
      </Layout>
    );
  }
}

export default App;
